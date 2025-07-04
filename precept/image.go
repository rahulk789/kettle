package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/containerd/containerd/pkg/kmutex"
	"github.com/containerd/containerd/v2/core/content"
	"github.com/containerd/containerd/v2/core/diff"
	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/containerd/v2/core/images/usage"
	"github.com/containerd/containerd/v2/core/snapshots"
	"github.com/containerd/containerd/v2/pkg/labels"
	"github.com/containerd/containerd/v2/pkg/rootfs"
	"github.com/containerd/errdefs"
	"github.com/containerd/platforms"
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/identity"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Image describes an image used by containers
type Image interface {
	// Name of the image
	Name() string
	// Target descriptor for the image content
	Target() ocispec.Descriptor
	// Labels of the image
	Labels() map[string]string
	// Unpack unpacks the image's content into a snapshot
	Unpack(context.Context, string, ...UnpackOpt) error
	// RootFS returns the unpacked diffids that make up images rootfs.
	RootFS(ctx context.Context) ([]digest.Digest, error)
	// Size returns the total size of the image's packed resources.
	Size(ctx context.Context) (int64, error)
	// Usage returns a usage calculation for the image.
	Usage(context.Context, ...UsageOpt) (int64, error)
	// Config descriptor for the image.
	Config(ctx context.Context) (ocispec.Descriptor, error)
	// IsUnpacked returns whether an image is unpacked.
	IsUnpacked(context.Context, string) (bool, error)
	// ContentStore provides a content store which contains image blob data
	ContentStore() content.Store
	// Metadata returns the underlying image metadata
	Metadata() images.Image
	// Platform returns the platform match comparer. Can be nil.
	Platform() platforms.MatchComparer
	// Spec returns the OCI image spec for a given image.
	Spec(ctx context.Context) (ocispec.Image, error)
}

type usageOptions struct {
	manifestLimit *int
	manifestOnly  bool
	snapshots     bool
}

// UsageOpt is used to configure the usage calculation
type UsageOpt func(*usageOptions) error

// WithUsageManifestLimit sets the limit to the number of manifests which will
// be walked for usage. Setting this value to 0 will require all manifests to
// be walked, returning ErrNotFound if manifests are missing.
// NOTE: By default all manifests which exist will be walked
// and any non-existent manifests and their subobjects will be ignored.
func WithUsageManifestLimit(i int) UsageOpt {
	// If 0 then don't filter any manifests
	// By default limits to current platform
	return func(o *usageOptions) error {
		o.manifestLimit = &i
		return nil
	}
}

// WithSnapshotUsage will check for referenced snapshots from the image objects
// and include the snapshot size in the total usage.
func WithSnapshotUsage() UsageOpt {
	return func(o *usageOptions) error {
		o.snapshots = true
		return nil
	}
}

// WithManifestUsage is used to get the usage for an image based on what is
// reported by the manifests rather than what exists in the content store.
// NOTE: This function is best used with the manifest limit set to get a
// consistent value, otherwise non-existent manifests will be excluded.
func WithManifestUsage() UsageOpt {
	return func(o *usageOptions) error {
		o.manifestOnly = true
		return nil
	}
}

var _ = (Image)(&image{})

// NewImage returns a client image object from the metadata image
func NewImage(client *Client, i images.Image) Image {
	return &image{
		client:   client,
		i:        i,
		platform: client.platform,
	}
}

// NewImageWithPlatform returns a client image object from the metadata image
func NewImageWithPlatform(client *Client, i images.Image, platform platforms.MatchComparer) Image {
	return &image{
		client:   client,
		i:        i,
		platform: platform,
	}
}

type image struct {
	client *Client

	i        images.Image
	platform platforms.MatchComparer
	diffIDs  []digest.Digest

	mu sync.Mutex
}

func (i *image) Metadata() images.Image {
	return i.i
}

func (i *image) Name() string {
	return i.i.Name
}

func (i *image) Target() ocispec.Descriptor {
	return i.i.Target
}

func (i *image) Labels() map[string]string {
	return i.i.Labels
}

func (i *image) RootFS(ctx context.Context) ([]digest.Digest, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.diffIDs != nil {
		return i.diffIDs, nil
	}

	provider := i.client.ContentStore()
	diffIDs, err := i.i.RootFS(ctx, provider, i.platform)
	if err != nil {
		return nil, err
	}
	i.diffIDs = diffIDs
	return diffIDs, nil
}

func (i *image) Size(ctx context.Context) (int64, error) {
	return usage.CalculateImageUsage(ctx, i.i, i.client.ContentStore(), usage.WithManifestLimit(i.platform, 1), usage.WithManifestUsage())
}

func (i *image) Usage(ctx context.Context, opts ...UsageOpt) (int64, error) {
	var config usageOptions
	for _, opt := range opts {
		if err := opt(&config); err != nil {
			return 0, err
		}
	}

	var usageOpts []usage.Opt
	if config.manifestLimit != nil {
		usageOpts = append(usageOpts, usage.WithManifestLimit(i.platform, *config.manifestLimit))
	}
	if config.snapshots {
		usageOpts = append(usageOpts, usage.WithSnapshotters(i.client.SnapshotService))
	}
	if config.manifestOnly {
		usageOpts = append(usageOpts, usage.WithManifestUsage())
	}

	return usage.CalculateImageUsage(ctx, i.i, i.client.ContentStore(), usageOpts...)
}

func (i *image) Config(ctx context.Context) (ocispec.Descriptor, error) {
	provider := i.client.ContentStore()
	return i.i.Config(ctx, provider, i.platform)
}

func (i *image) IsUnpacked(ctx context.Context, snapshotterName string) (bool, error) {
	sn, err := i.client.getSnapshotter(ctx, snapshotterName)
	if err != nil {
		return false, err
	}

	diffs, err := i.RootFS(ctx)
	if err != nil {
		return false, err
	}

	if _, err := sn.Stat(ctx, identity.ChainID(diffs).String()); err != nil {
		if errdefs.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (i *image) Spec(ctx context.Context) (ocispec.Image, error) {
	var ociImage ocispec.Image

	desc, err := i.Config(ctx)
	if err != nil {
		return ociImage, fmt.Errorf("get image config descriptor: %w", err)
	}

	blob, err := content.ReadBlob(ctx, i.ContentStore(), desc)
	if err != nil {
		return ociImage, fmt.Errorf("read image config from content store: %w", err)
	}

	if err := json.Unmarshal(blob, &ociImage); err != nil {
		return ociImage, fmt.Errorf("unmarshal image config %s: %w", blob, err)
	}

	return ociImage, nil
}

// UnpackConfig provides configuration for the unpack of an image
type UnpackConfig struct {
	// ApplyOpts for applying a diff to a snapshotter
	ApplyOpts []diff.ApplyOpt
	// SnapshotOpts for configuring a snapshotter
	SnapshotOpts []snapshots.Opt
	// CheckPlatformSupported is whether to validate that a snapshotter
	// supports an image's platform before unpacking
	CheckPlatformSupported bool
	// DuplicationSuppressor is used to make sure that there is only one
	// in-flight fetch request or unpack handler for a given descriptor's
	// digest or chain ID.
	DuplicationSuppressor kmutex.KeyedLocker
}

// UnpackOpt provides configuration for unpack
type UnpackOpt func(context.Context, *UnpackConfig) error

// WithSnapshotterPlatformCheck sets `CheckPlatformSupported` on the UnpackConfig
func WithSnapshotterPlatformCheck() UnpackOpt {
	return func(ctx context.Context, uc *UnpackConfig) error {
		uc.CheckPlatformSupported = true
		return nil
	}
}

// WithUnpackDuplicationSuppressor sets `DuplicationSuppressor` on the UnpackConfig.
func WithUnpackDuplicationSuppressor(suppressor kmutex.KeyedLocker) UnpackOpt {
	return func(ctx context.Context, uc *UnpackConfig) error {
		uc.DuplicationSuppressor = suppressor
		return nil
	}
}

// WithUnpackApplyOpts appends new apply options on the UnpackConfig.
func WithUnpackApplyOpts(opts ...diff.ApplyOpt) UnpackOpt {
	return func(ctx context.Context, uc *UnpackConfig) error {
		uc.ApplyOpts = append(uc.ApplyOpts, opts...)
		return nil
	}
}

func (i *image) Unpack(ctx context.Context, snapshotterName string, opts ...UnpackOpt) error {
	ctx, done, err := i.client.WithLease(ctx)
	if err != nil {
		return err
	}
	defer done(ctx)

	var config UnpackConfig
	for _, o := range opts {
		if err := o(ctx, &config); err != nil {
			return err
		}
	}

	manifest, err := i.getManifest(ctx, i.platform)
	if err != nil {
		return err
	}

	layers, err := i.getLayers(ctx, manifest)
	if err != nil {
		return err
	}

	var (
		a  = i.client.DiffService()
		cs = i.client.ContentStore()

		chain    []digest.Digest
		unpacked bool
	)
	snapshotterName, err = i.client.resolveSnapshotterName(ctx, snapshotterName)
	if err != nil {
		return err
	}
	sn, err := i.client.getSnapshotter(ctx, snapshotterName)
	if err != nil {
		return err
	}
	if config.CheckPlatformSupported {
		if err := i.checkSnapshotterSupport(ctx, snapshotterName, manifest); err != nil {
			return err
		}
	}

	for _, layer := range layers {
		unpacked, err = rootfs.ApplyLayerWithOpts(ctx, layer, chain, sn, a, config.SnapshotOpts, config.ApplyOpts)
		if err != nil {
			return fmt.Errorf("apply layer error for %q: %w", i.Name(), err)
		}

		if unpacked {
			// Set the uncompressed label after the uncompressed
			// digest has been verified through apply.
			cinfo := content.Info{
				Digest: layer.Blob.Digest,
				Labels: map[string]string{
					labels.LabelUncompressed: layer.Diff.Digest.String(),
				},
			}
			if _, err := cs.Update(ctx, cinfo, "labels."+labels.LabelUncompressed); err != nil {
				return err
			}
		}

		chain = append(chain, layer.Diff.Digest)
	}

	desc, err := i.i.Config(ctx, cs, i.platform)
	if err != nil {
		return err
	}

	rootFS := identity.ChainID(chain).String()

	cinfo := content.Info{
		Digest: desc.Digest,
		Labels: map[string]string{
			fmt.Sprintf("containerd.io/gc.ref.snapshot.%s", snapshotterName): rootFS,
		},
	}

	_, err = cs.Update(ctx, cinfo, fmt.Sprintf("labels.containerd.io/gc.ref.snapshot.%s", snapshotterName))
	return err
}

func (i *image) getManifest(ctx context.Context, platform platforms.MatchComparer) (ocispec.Manifest, error) {
	cs := i.ContentStore()
	manifest, err := images.Manifest(ctx, cs, i.i.Target, platform)
	if err != nil {
		return ocispec.Manifest{}, err
	}
	return manifest, nil
}

func (i *image) getLayers(ctx context.Context, manifest ocispec.Manifest) ([]rootfs.Layer, error) {
	diffIDs, err := i.RootFS(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve rootfs: %w", err)
	}

	// parse out the image layers from oci artifact layers
	imageLayers := []ocispec.Descriptor{}
	for _, ociLayer := range manifest.Layers {
		if images.IsLayerType(ociLayer.MediaType) {
			imageLayers = append(imageLayers, ociLayer)
		}
	}
	if len(diffIDs) != len(imageLayers) {
		return nil, errors.New("mismatched image rootfs and manifest layers")
	}
	layers := make([]rootfs.Layer, len(diffIDs))
	for i := range diffIDs {
		layers[i].Diff = ocispec.Descriptor{
			// TODO: derive media type from compressed type
			MediaType: ocispec.MediaTypeImageLayer,
			Digest:    diffIDs[i],
		}
		layers[i].Blob = imageLayers[i]
	}
	return layers, nil
}

func (i *image) checkSnapshotterSupport(ctx context.Context, snapshotterName string, manifest ocispec.Manifest) error {
	snapshotterPlatformMatcher, err := i.client.GetSnapshotterSupportedPlatforms(ctx, snapshotterName)
	if err != nil {
		return err
	}

	manifestPlatform, err := images.ConfigPlatform(ctx, i.ContentStore(), manifest.Config)
	if err != nil {
		return err
	}

	if snapshotterPlatformMatcher.Match(manifestPlatform) {
		return nil
	}
	return fmt.Errorf("snapshotter %s does not support platform %s for image %s", snapshotterName, manifestPlatform, manifest.Config.Digest)
}

func (i *image) ContentStore() content.Store {
	return i.client.ContentStore()
}

func (i *image) Platform() platforms.MatchComparer {
	return i.platform
}
