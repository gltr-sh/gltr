***NOTE: WORKFLOW HAS CHANGED RECENTLY - THESE INSTRUCTIONS ARE NOW OUTDATED***

`glattr` is a lightweight tool to support low friction data analysis. It enables
local and remote work to be done in a seamless fashion, integrating compute
and data workflows conveniently.

# Building `glattr`

```
go build
```

# Initializing `glattr`

Currently, `glattr` only works with AWS; more specifically, it works with two
AWS services - ECS and EC2 - to run data science workloads on these services.

It is necessary to have an AWS account with sufficient privilege to create the
basic resources - these are primarily networking resources so VPC authorizations
are required. (To date, most of the work has been done using admin privileges
as this reflects the initial use case). Assuming an AWS account has been
configured properly and credentials are available (typically in `$HOME/aws`),
then `glattr` can be initialized as follows:

```
glattr init --aws
```

This will create resources necessary for `glattr` to run, including a VPC,
subnet, internet gateway and appropriate routes. Note that the VPC setup is
deliberately not a HA configuration; rather it's a simple configuration which
can be removed easily as necessary. An ECS cluster is also created.

The initialization phase does not create any resources which have direct
cost implications, ie these resources can exist but costs are only incurred
when they are used.

# Initializing a project

When `glattr` has been initialized, it is then possible to initialize a
project - project initialization specifies what compute platform is used
and what container images are used; project initialization also configures
security groups for port connectivity.

Initialize a project as follows:
```
glattr init-project
```

You will then have to answer a set of questions regarding how your project
will work.

# Running the project

Once the project has been initialized, it is possible to run the project using
```
glattr run
```

If the project has been configured correctly, the project will be run on the
execution platform chosen. Note that this can result in consumption of AWS
costs.

# Listing tasks

```
glattr list-tasks
```

# Removing tasks

```
glattr kill-task --task-id <task-id>
```

# Removing everything

```
glattr powerhose --aws
```

This will remove all `glattr` resources which have been used. Note that this
can fail if there are some network connections still in use; in the case that
an ECS tasks has just been terminated, it can take some minutes before all
state is removed - hence the cleanup operation can fail. Wait for some time
and run it again and the cleanup should be successful.
