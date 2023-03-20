aws iam create-role --role-name ecs-role --assume-role-policy-document file://assume-role.json
aws iam put-role-policy --role-name ecs-role --policy-name ecsAccessPolicy --policy-document file://ecs-execution-role.json
