# create cluster
echo
echo "Deleting task..."
aws ecs stop-task --cluster test-cluster --task sleep360

echo
echo "DeRegistering task definition..."
aws ecs deregister-task-definition --cli-input-json file://sleep360.json

echo "Removing cluster..."
aws ecs delete-cluster --cluster test-cluster 


    




