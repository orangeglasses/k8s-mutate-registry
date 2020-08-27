# k8s-mutate-registry

This webhook overrules the configured image locations in pods. A mapping table is used to map between public registries and an internal url
deplyoment and configuration

## Deploying
Use the deploy/deploy.sh script in this repo. This script is testen with VMware TKGI 1.8.1 (formerly known as PKS).

### Configuration
When running the script a configure json needs to be specified. The config json should follow this format:

{

  "defaultDomain": "registry-1.docker.io",
  "domainMapping": {
    "registry-1.docker.io": "internal-docker.example.internal",
    "quay.io": "internal-quay.example.internal"
  }
}

 - defaultDomain is optional. If no default domain is given then registry-1.docker.io will be used as the default domain.
 - the defaultDomain should ALWAYS be present in the domainMapping as well. Even if you omit th edefaultdomain setting you should put registry-1.docker.io in the domainMapping
 - The domain Mapping is simply a key/value list of externel and internal registry domain names. In the above example, if you pull an image from quay.io; let's say quaiy.io/ubuntu then the image will be overwritten as internal-quay.example.internal/ubunt
 
 ### namespace labels
 The webhook config that comes in the deploy folder will only be active on namespaces with a label: <app name>: enabled. <app name> is the app name you configured when running the deploy.sh script.
