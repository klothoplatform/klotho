from . import fs as s3fs
import boto3
import os
import logging
import json
import uuid

lambda_client = boto3.client("lambda")
APP_NAME = os.environ.get("APP_NAME")


async def proxy_call(exec_group_name, module_name, function_name, params):
    payload_key = str(uuid.uuid4())
    async with s3fs.open(payload_key, mode='w') as f:
        await f.write(json.dumps(params))
    physical_address = get_exec_unit_lambda_function_name(exec_group_name)
    payload_to_send = {
        "module_name": module_name,
        "function_to_call": function_name,
        "params": payload_key,
    }
    result = lambda_client.invoke(
        FunctionName=physical_address,
        Payload=json.dumps(payload_to_send))
    dispatcher_param_key_result = json.load(result["Payload"])
    async with s3fs.open(dispatcher_param_key_result) as f:
        return json.loads(await f.read())


def get_exec_unit_lambda_function_name(logical_name):
    return f'{APP_NAME}-{logical_name}'
