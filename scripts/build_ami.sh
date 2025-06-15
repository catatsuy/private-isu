#!/usr/bin/env bash
set -euo pipefail

# This script launches an EC2 instance, runs Ansible to configure it,
# creates an AMI from the instance, and outputs the AMI ID.

BASE_AMI=${BASE_AMI:?BASE_AMI is required}
INSTANCE_TYPE=${INSTANCE_TYPE:-t2.micro}
SSH_KEY_NAME=${SSH_KEY_NAME:?SSH_KEY_NAME is required}
SECURITY_GROUP=${SECURITY_GROUP:?SECURITY_GROUP is required}
SUBNET_ID=${SUBNET_ID:?SUBNET_ID is required}
SSH_PRIVATE_KEY=${SSH_PRIVATE_KEY:?SSH_PRIVATE_KEY is required}

INSTANCE_JSON=$(mktemp)
aws ec2 run-instances \
  --image-id "$BASE_AMI" \
  --count 1 \
  --instance-type "$INSTANCE_TYPE" \
  --key-name "$SSH_KEY_NAME" \
  --security-group-ids "$SECURITY_GROUP" \
  --subnet-id "$SUBNET_ID" \
  > "$INSTANCE_JSON"

INSTANCE_ID=$(jq -r '.Instances[0].InstanceId' "$INSTANCE_JSON")
aws ec2 wait instance-running --instance-ids "$INSTANCE_ID"

PUBLIC_IP=$(aws ec2 describe-instances --instance-ids "$INSTANCE_ID" \
  --query 'Reservations[0].Instances[0].PublicIpAddress' --output text)

export ANSIBLE_HOST_KEY_CHECKING=False

cat <<EOF2 > inventory
$PUBLIC_IP ansible_user=ubuntu ansible_ssh_private_key_file=$SSH_PRIVATE_KEY
EOF2

ansible-playbook -i inventory provisioning/image/ansible/playbooks.yml --skip-tags nodejs

AMI_ID=$(aws ec2 create-image --instance-id "$INSTANCE_ID" \
  --name "private-isu-$(date +%Y%m%d%H%M%S)" \
  --query 'ImageId' --output text)

aws ec2 wait image-available --image-ids "$AMI_ID"

aws ec2 terminate-instances --instance-ids "$INSTANCE_ID"

# Output AMI ID for GitHub Actions
echo "ami_id=$AMI_ID" >> "$GITHUB_OUTPUT"
