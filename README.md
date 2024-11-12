# Mirantis Launchpad

Mirantis Launchpad CLI tool ("**launchpad**") simplifies and automates deploying [Mirantis Container Runtime](https://docs.mirantis.com/welcome/mcr), [Mirantis Kubernetes Engine](https://docs.mirantis.com/welcome/mke) and [Mirantis Secure Registry](https://docs.mirantis.com/welcome/msr) on public clouds (like AWS or Azure), private clouds (like OpenStack or VMware), virtualization platforms (like VirtualBox, VMware Workstation, Parallels, etc.), or bare metal.

Launchpad can also provide full cluster lifecycle management. Multi-manager, high availability clusters, defined as having sufficient node capacity to move active workloads around while updating, can be upgraded with no downtime.

## Documentation

Launchpad documentation can be browsed on the [Mirantis Documentation site](https://docs.mirantis.com/mke/3.7/launchpad.html).

## Example

Launchpad reads a YAML configuration file which lists cluster hosts with their connection addresses and product settings. It will then connect to each of the hosts, make the necessary preparations and finally install, upgrade or uninstall the cluster to match the desired state.

An example configuration:

```yaml
apiVersion: launchpad.mirantis.com/mke/v1.3
kind: mke
spec:
  hosts:
    - role: manager
      ssh:
        address: 10.0.0.1
        user: root
    - role: worker
      ssh:
        address: 10.0.0.2
        user: ubuntu
  mke:
    version: 3.7.7
```

Installing a cluster:

```
$ launchpad apply --config launchpad.yaml


                       ..,,,,,..
              .:i1fCG0088@@@@@880GCLt;,               .,,::::::,,...
         ,;tC0@@@@@@@@@@@@@@@@@@@@@@@@@0:,     .,:ii111i;:,,..
      ,;1ttt1;;::::;;itfCG8@@@@@@@@@i @@@@0fi1t111i;,.
     .,.                  .:1L0@@   @8GCft111ii1;
                               :f0CLft1i;i1tL . @8Cti:.               .,:,.
                           .:;i1111i;itC;  @@@@@@@@@@@80GCLftt11ttfLLLf1:.
                    .,:;ii1111i:,.    , G8@@@@@@@@@@@@@@@@@@@@@@@0Lt;,
            ...,,::;;;;::,.               ,;itfLCGGG0GGGCLft1;:.



   ;1:      i1, .1, .11111i:      .1i     :1;     ,1, i11111111: ;i   ;1111;
   G@GC:  1G0@i ;@1 ;@t:::;G0.   .0G8f    L@GC:   i@i :;;;@G;;;, C@ .80i:,:;
   C8 10CGC::@i :@i :@f:;;;CG.  .0G ,@L   f@.iGL, ;@;     @L     L@. tLft1;.
   G8   1;  ;@i ;@i :@L11C@t   ,08fffL@L  L@.  10fi@;    .@L     L@.    .:t@1
   C0       ;@i :@i :@i   ;Gf..0C     ,8L f@.   .f0@;    .8L     L8  fft11fG;
   ..        .   .   ..     ,..,        , ..      ..      ..     ..  .,:::,

   Mirantis Launchpad (c) 2021 Mirantis, Inc.

INFO ==> Running phase: Open Remote Connection
INFO ==> Running phase: Detect host operating systems
INFO [ssh] 10.0.0.2:22: is running Ubuntu 18.04.5 LTS
INFO [ssh] 10.0.0.1:22: is running Ubuntu 18.04.5 LTS
INFO ==> Running phase: Gather Facts
INFO [ssh] 10.0.0.1:22: gathering host facts
INFO [ssh] 10.0.0.2:22: gathering host facts
INFO [ssh] 10.0.0.1:22: internal address: 172.17.0.2
INFO [ssh] 10.0.0.1:22: gathered all facts
INFO [ssh] 10.0.0.2:22: internal address: 172.17.0.3
INFO [ssh] 10.0.0.2:22: gathered all facts
...
...
INFO Cluster is now configured.  You can access your admin UIs at:
INFO MKE cluster admin UI: https://test-mke-cluster.example.com
INFO You can also download the admin client bundle with the following command: launchpad client-config
```

## Support, Reporting Issues & Feedback

Please use Github [issues](https://github.com/Mirantis/launchpad/issues) to report any issues, provide feedback, or request support.
