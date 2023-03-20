# add this for debug mode...
#  --debug \

#IMAGE=jupyter/scipy-notebook:85f615d5cafa 
# IMAGE=public.ecr.aws/bitnami/jupyter-base-notebook:latest 
IMAGE=gltr/jn:with-js
# IMAGE=gltr/base-container

# add command if required....
# ping -c 5 google.com

ecs-cli run --cluster test-cluster \
  --public \
  --env "JUPYTERHUB_API_TOKEN=4ae392cdf3d27c4002e6006caf9267408ca414b7b7070685e09247cd40646b3a" \
  -p "8888:8888" \
  -p "22:22" \
  --fargate --cpu-reservation 1024  \
  --role arn:aws:iam::615311330244:role/ecs-role \
  --subnet-filter "tag:Name=project-subnet-public1-eu-west-1a" \
  --security-groups "launch-wizard-3" \
  $IMAGE # jupyter lab --ip 0.0.0.0
