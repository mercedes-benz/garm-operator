#!/bin/bash

# this script ca be used to create a vault kubernetes auth role
# the role is created with the following naming pattern: <NAMESPACE_NAME>_<SERVICEACCOUNT_NAME>

# help function to show availible options
help()
{
   echo "Syntax: k8s_auth.sh [-h|s|n|p|r|c|f|d]"
   echo "options:"
   echo "-h     Print this Help"
   echo "-s     Set Name of Serviceaccount (required)"
   echo "-n     Set Namespace of Serviceaccount (required)"
   echo "-p     Set vault auth path (required)"
   echo "-r     Set vault role policy (required)"
   echo "-c     Set Name of Kubernetes Cluster (only required if argument -f is not set)"
   echo "-f     Skip any checks for the auth method, only create the role"
   echo "-d     Set to delete the Vault Role"
}

# parse provided options and arguments
while getopts ":hs:n:p:r:c:fd" option; do
  case $option in
    h) # display help
      help
      exit;;
    f) # skip any checks for the auth method, only create the role
      ONLY_CREATE_ROLE=true
      ;;
    s) # set VAULT_SERVICEACCOUNT_NAME var
      if [ "${OPTARG:0:1}" == "-" ]; then
        echo "Error: Option -$option starts with option string $OPTARG"
        exit 1
      fi
      VAULT_SERVICEACCOUNT_NAME=$OPTARG
      ;;
    n) # set VAULT_SERVICEACCOUNT_NAMESPACE var
      if [ "${OPTARG:0:1}" == "-" ]; then
        echo "Error: Option -$option starts with option string $OPTARG"
        exit 1
      fi
      VAULT_SERVICEACCOUNT_NAMESPACE=$OPTARG
      ;;
    p) # set VAULT_AUTH_PATH var
      if [ "${OPTARG:0:1}" == "-" ]; then
        echo "Error: Option -$option starts with option string $OPTARG"
        exit 1
      fi
      VAULT_AUTH_PATH=$OPTARG
      ;;
    r) # set VAULT_ROLE_POLICY var
      if [ "${OPTARG:0:1}" == "-" ]; then
        echo "Error: Option -$option starts with option string $OPTARG"
        exit 1
      fi
      VAULT_ROLE_POLICY=$OPTARG
      ;;
    c) # set KUBE_CLUSTER_NAME var
      if [ "${OPTARG:0:1}" == "-" ]; then
        echo "Error: Option -$option starts with option string $OPTARG"
        exit 1
      fi
      KUBE_CLUSTER_NAME=$OPTARG
      ;;
    d) # set VAULT_DELETE_ROLE var
      VAULT_DELETE_ROLE=true
      ;;
    \?) # check if provided option is availible
      echo "Error: Invalid option"
      echo "Use -h option to see all availible arguments"
      exit;;
    :) # check if provided option has an argument
      echo "Error: Option -$OPTARG requires an argument"
      exit;;
  esac
done

# check if -d is set
if [[ -z $VAULT_DELETE_ROLE ]];then
  if [[ -z $VAULT_SERVICEACCOUNT_NAME || -z $VAULT_SERVICEACCOUNT_NAMESPACE || -z $VAULT_AUTH_PATH || -z $VAULT_ROLE_POLICY ]];then
    echo "Error: One or more of the following required arguments are not set: -s, -n, -p, -r"
    help
    exit 1
  fi
else
  if [[ -z $VAULT_SERVICEACCOUNT_NAME || -z $VAULT_SERVICEACCOUNT_NAMESPACE || -z $VAULT_AUTH_PATH ]];then
    echo "Error: One or more of the following required arguments are not set: -s, -n, -p"
    help
    exit 1
  fi
fi

# when -f and -d is not set
if [[ -z $ONLY_CREATE_ROLE && -z $VAULT_DELETE_ROLE ]];then
  if [[ -z $KUBE_CLUSTER_NAME ]];then
    echo "Error: Argument -c is required"
    help
    exit 1
  fi

  VAULT_AUTH_LIST=$(vault auth list -format json | jq -r '."'$VAULT_AUTH_PATH/'".type')
  KUBE_HOST=$(kubectl config view --raw -o 'jsonpath={.clusters[?(@.name == "'$KUBE_CLUSTER_NAME'")].cluster.server}')
  KUBE_CA_CERT=$(kubectl config view --raw -o jsonpath='{.clusters[?(@.name == "'$KUBE_CLUSTER_NAME'")].cluster.certificate-authority-data}' | base64 --decode)

  if [ $VAULT_AUTH_LIST == 'null' ];then
    echo "Vault kubernetes auth will be enabled on path $VAULT_AUTH_PATH"
    vault auth enable -path $VAULT_AUTH_PATH kubernetes
    # this configures vault's k8s auth method so that the same token that is used to login
    # can be used to authorize it's own token review in k8s
    # by this we don't need some secondary token review jwt (which we would have to care for)
    # see: https://developer.hashicorp.com/vault/api-docs/auth/kubernetes#configure-method
    vault write auth/$VAULT_AUTH_PATH/config \
      kubernetes_host="$KUBE_HOST" \
      kubernetes_ca_cert="$KUBE_CA_CERT" \
      token_reviewer_jwt="" \
      disable_local_ca_jwt="true"
  else
    echo "Vault kubernetes auth is already enabled on path $VAULT_AUTH_PATH"
  fi
fi

# TODO:
# - do we want to split prod / int ...
# - probably we need to set more fields for solid security .. e.g. token_max_ttl, see :
#   https://developer.hashicorp.com/vault/api-docs/auth/kubernetes#create-role
# Important:
# - we need to set the alias_name_source to 'serviceaccount_name' otherwise it will use 'serviceaccount_uid'
#   and this will create a new client for each deployment of the service account, and this will make the
#   license fee counter +1
vault `if [[ -z $VAULT_DELETE_ROLE ]]; then echo "write"; else echo "delete"; fi` \
  auth/$VAULT_AUTH_PATH/role/$VAULT_SERVICEACCOUNT_NAMESPACE'_'$VAULT_SERVICEACCOUNT_NAME `if [[ -z $VAULT_DELETE_ROLE ]]; then echo "\
  bound_service_account_names=$VAULT_SERVICEACCOUNT_NAME \
  bound_service_account_namespaces=$VAULT_SERVICEACCOUNT_NAMESPACE \
  alias_name_source=serviceaccount_name \
  policies=$VAULT_ROLE_POLICY \
  token_no_default_policy=true \
  ttl=10m"
  fi`

# ./hack/vault-k8s-auth.sh -s garm-operator-controller-manager -n garm-infra-stage-int -p kubernetes -r cicd -f
