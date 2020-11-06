#!/bin/bash

DEFAULT_IMAGE="vchrisr/k8smutateregistry:latest"

usage () {
echo -e "usage: $0 -i <docker image> -n <namespace> -c <config file>
description: 
    Deploys the k8s-mutate-registry app and webhook configuration

options:    
    -i [image]             The docker image to use (defaults to: $DEFAULT_IMAGE) 
    -n [namespace]         Namespace the app will be deployed to
    -a [app name]          App name to use
    -c [config file]       config json file. see config-example.json
"
}
 
while getopts i:n:a:c: flag
do
  case $flag in    
    i) IMAGE=$OPTARG;;
    n) NAMESPACE=$OPTARG;;    
    a) APP=$OPTARG;;    
    c) CONFIG_FILE=$OPTARG;;    
    *) usage ; exit 1;;
  esac
done
shift $(( $OPTIND -1))

if [[ -z $IMAGE ]]; then
  echo "Image not defined. Using default: ${DEFAULT_IMAGE}"
  IMAGE=$DEFAULT_IMAGE  
fi

if [[ -z $NAMESPACE ]]; then
  echo "No namespace defined"
  usage
  exit 1
fi

if [[ -z $CONFIG_FILE ]]; then
  echo "No config file defined"
  usage
  exit 1
fi

if [[ ! -f $CONFIG_FILE ]]; then
  echo "Config file not found"
  usage
  exit 1
fi

FULLPATH=$(dirname  "$0")
CONFIG_JSON=$(cat $CONFIG_FILE | tr -d '\n' | tr -d ' ')

echo "Creating TLS cert... " 
$FULLPATH/ssl.sh -a $APP -n $NAMESPACE

echo "Parsing template..."
YAML="$(mktemp)"

export IMAGE
export CA_BUNDLE=$(cat ${APP}.cabundle)
export CONFIG_JSON
export NAMESPACE
export APP
envsubst '${IMAGE} ${CA_BUNDLE} ${CONFIG_JSON} ${NAMESPACE}, ${APP}' <k8s-mutate-registry.yml > $YAML

echo "Applying yaml..."
kubectl apply -f $YAML