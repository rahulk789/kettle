# kettle
![image](https://raw.githubusercontent.com/rahulk789/kettle/refs/heads/main/assets/kettle.png)
<sup><sub>
___Reminder to self___  
Okay, so what am I making? What I expect this binary to do is to take requests from the user to spawn x amount of runc containers, delete them when necessary as suggested by the user. When the runc container crashes, I want you to manage it. I want you to record the logs, help the user copy and store files to and from the runc container.
So there are several ways to create a container. you can either directly start a shim manually which creates a container using runc or you can have a containerd like service running which creates the shim but we need ctr to interact with these things. we could possibly have ctr functionality for now inside shim itself so that we can start and stop one container at a time
You need to remember that just creating a container "as per user" or deleting it could be done by doing exec.Command( runc create container).Your project is more than just this and must be able to detect crashloopbackoffs and try creating the container again.
Now how would you do this? do you need a server? I know we need a running process to manage this aka the shim  
kctl calls task.create to ttrpc server run by shim which calls runc with flags to create container in bundle after I manually create rootfs dir and cp sh file into bin
just do sudo runc list to get the container name
how do i manage this from containerd level though. do i have multiple ttrpc servers? probably not since only one socket path is there. probably multiple clients having one server which spawns multiple processes in the form of a container service.
