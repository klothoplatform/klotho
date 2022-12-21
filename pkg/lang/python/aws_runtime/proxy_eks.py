import boto3
import os
import logging
import requests

sd_client = boto3.client("servicediscovery")
APP_NAME = os.environ.get("APP_NAME")


async def proxy_call(exec_group_name, module_name, function_to_call, params):
    try:
        hostname = get_exec_fargate_instance(exec_group_name)
        res = requests.post(f'http://{hostname}:3001', json={
            'exec_group_name': exec_group_name,
            'function_to_call': function_to_call,
            'module_name': module_name,
            'params': params,
        })
        return res.content
    except Exception as e:
        logging.error(e)
        raise e


def get_exec_fargate_instance(logical_name):
    response = sd_client.discover_instances(
        NamespaceName='default',  # ECS uses an app-specific name, but for EKS it's just "default"
        ServiceName=logical_name,
    )
    ips = [ip["Attributes"]["AWS_INSTANCE_IPV4"] for ip in response['Instances']]
    if len(ips) == 0:
        raise Exception(f'No IPs found for {logical_name}')
    return ips[0]
