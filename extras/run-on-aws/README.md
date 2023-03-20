# What is this

This is a simple demo which shows how a docker container can be built and
run on a cloud service easily.

It build a docker container which does the following:
- clones a public git repo
- runs an ssh daemon
- runs jupyter lab

This can be run locally and it can be run on ECS.

Notes:
- still loads of stuff to do here
- ssh server not v well set up - no authorized keys injected yet
- jupyter lab is not running in right place with right privileges yet
- git clone only clones public repo - no cloning of private repo supported
- v untested
- the ecs-cli.sh script is handy for testing stuff out, but basically needs to be rewritten for our context
- it assumes that there has been quite a bit of stuff set up a priori
