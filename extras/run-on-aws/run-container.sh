# create cluster
echo "Creating cluster..."
aws ecs create-cluster --cluster-name test-cluster --capacity-provider FARGATE

# ( aws ecs delete-cluster --cluster test-cluster )

# create task definition

echo
echo "Registering task definition..."
aws ecs register-task-definition \
    --cli-input-json file://sleep360.json
    
# run task
echo
echo "Running task..."
aws ecs run-task --cluster test-cluster --task-definition sleep360:1




