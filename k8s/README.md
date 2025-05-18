# Deploying Meds-Bot to k3s

This guide explains how to deploy the Medication Reminder Discord Bot to a k3s Kubernetes cluster.

## Prerequisites

- A running k3s cluster
- kubectl configured to access your k3s cluster
- Docker installed for building the container image
- Git to clone the repository

## Deployment Steps

### 1. Build the Docker Image

```bash
# Clone the repository if you haven't already
git clone https://github.com/yourusername/meds-bot.git
cd meds-bot

# Build the Docker image
docker build -t meds-bot:latest .

# If using a remote registry, tag and push the image
# docker tag meds-bot:latest your-registry/meds-bot:latest
# docker push your-registry/meds-bot:latest
```

### 2. Configure the Deployment

Edit the Kubernetes configuration files in the `k8s` directory:

1. Update `secret.yaml` with your Discord token and user ID:
   ```yaml
   stringData:
     DISCORD_TOKEN: "your-discord-token-here"
     DISCORD_USER_ID_TO_PING: "your-user-id-here"
   ```

2. Update `configmap.yaml` with your Discord channel ID and medication settings:
   ```yaml
   data:
     DISCORD_CHANNEL_ID: "your-channel-id-here"
     # Customize other settings as needed
   ```

3. If using a remote registry, uncomment and update the image section in `kustomization.yaml`:
   ```yaml
   images:
   - name: meds-bot
     newName: your-registry/meds-bot
     newTag: latest
   ```

### 3. Deploy to k3s

```bash
# Apply all resources using kustomize
kubectl apply -k k8s/

# Check the deployment status
kubectl get pods -l app=meds-bot
kubectl get pvc meds-bot-data
```

### 4. View Logs

```bash
# Get the pod name
POD_NAME=$(kubectl get pods -l app=meds-bot -o jsonpath="{.items[0].metadata.name}")

# View logs
kubectl logs $POD_NAME
```

## Updating the Deployment

To update the deployment with a new version:

1. Build a new Docker image with a new tag
2. Update the image tag in `kustomization.yaml`
3. Apply the changes:
   ```bash
   kubectl apply -k k8s/
   ```

## Troubleshooting

### Pod is not starting

Check the pod status and logs:
```bash
kubectl describe pod -l app=meds-bot
kubectl logs -l app=meds-bot
```

### Database issues

If you need to access the database directly:
```bash
kubectl exec -it $POD_NAME -- sh
cd /app/data
sqlite3 meds_reminder.db
```

### Restarting the deployment

```bash
kubectl rollout restart deployment meds-bot
```

## Cleanup

To remove the deployment:
```bash
kubectl delete -k k8s/
```

Note: This will not delete the PersistentVolumeClaim by default to prevent data loss. To delete it:
```bash
kubectl delete pvc meds-bot-data
```