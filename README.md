# kettle
Kettle is a minimal wacky container runtime engine written to mimic a containerd like system for performing elementary actions on containers. Currently there is nothing custom about this project. I have made sure not to use any of containerd's out of the box functions to make my life easier (because where's the fun in that XD) although I will be using libcontainer, runc and other building blocks of containerd (I swear I will rewrite libcontainer too). The plan is to understand how this thing works and eventually make a tweak to it (like kata for example) to support a niche usecase (probably something gpu/ai training related but we already have beta9). If you're reading this, you're an awesome person.  

![image](https://github.com/user-attachments/assets/87e78fdf-a3f3-4528-9599-4f8f2bb80b46)
<sup><sub>
___Reminder to self___  
Okay, so what am I making? What I expect this binary to do is to take requests from the user to spawn x amount of runc containers, delete them when necessary as suggested by the user. When the runc container crashes, I want you to manage it. I want you to record the logs, help the user copy and store files to and from the runc container.
So there are several ways to create a container. you can either directly start a shim manually which creates a container using runc or you can have a containerd like service running which creates the shim but we need ctr to interact with these things. we could possibly have ctr functionality for now inside shim itself so that we can start and stop one container at a time
You need to remember that just creating a container "as per user" or deleting it could be done by doing exec.Command( runc create container).Your project is more than just this and must be able to detect crashloopbackoffs and try creating the container again.
Now how would you do this? do you need a server? I know we need a running process to manage this aka the shim  
kctl calls task.create to ttrpc server run by shim which calls runc with flags to create container in bundle after I manually create rootfs dir and cp sh file into bin
just do sudo runc list to get the container name
how do i manage this from containerd level though. do i have multiple ttrpc servers? probably not since only one socket path is there. probably multiple clients having one server which spawns multiple processes in the form of a container service.
We have ttrpc servers only as a  part of shims (not sure if i said it earlier). Now I need to figure out a method to know whether a shim for a given container id already exists or not and if not only then create the shim. Therefore kctl will call Start on containerd grpc to run a process and we follow that by figuring out if we need shim and if so do we just call a function or is there another proto api involved? because i see services/task api and services/containers api, whats the exact difference between these two proto? [Ans: we basically create a shim process through the binary. this binary inturn spins up a ttrpc server which has "task" based proto funcs. we do NOT use ttrpc to create the shim. we use ttrpc client only after creating the shim]
Test out the ctr -> shim -> run workflow. It should work fine? Debug logic after testing. Need to handle logic regarding checking the existance of shim before running the task on container. Read about how shim lifecycle is managed when a task scheduled has failed. Read about how containerd shim manages multiple runtimes for creating containers and how it defaults to runc alone 

![image](https://raw.githubusercontent.com/rahulk789/kettle/refs/heads/main/assets/kettle.png)


