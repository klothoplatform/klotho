from . import fs_payload as s3fs
import asyncio
import json
import logging
import os
import types
import uuid
import inspect

log_level = os.getenv("KLOTHO_LOG_LEVEL", "DEBUG").upper()

logging.basicConfig()
log = logging.getLogger("klotho")
log.setLevel(logging.DEBUG)

asgi_handler = None


def handler(event, context):
    request_handler = get_handler(event)
    if not request_handler:
        raise Exception("this request could not be handled: no handler found")

    result = request_handler(event, context)
    if isinstance(result, types.CoroutineType):
        result = asyncio.run(result)
    return result


def init_asgi_handler():
    global asgi_handler

    entrypoint = try_import("{{.Expose.AppModule}}")

    if entrypoint is None:
        raise Exception("startup failed: no entrypoint found")

    fastapi = try_import("fastapi")
    mangum = try_import("mangum")
    if not fastapi or not mangum:
        return

    app = get_fastapi_app(fastapi, entrypoint)
    asgi_handler = mangum.Mangum(app)
    return asgi_handler


async def rpc_handler(event, _context):
    payload_key = event.get('params')
    async with s3fs.open(payload_key) as f:
        params = json.loads(await f.read())
    module_obj = try_import(event.get('module_name'))
    if not module_obj:
        raise Exception("couldn't find module for path: {module_path}")
    function = getattr(module_obj, event.get('function_to_call'))
    param_args = ()
    param_kwargs = {}
    args_spec = inspect.getfullargspec(function)
    if args_spec.varkw:
        param_kwargs, params = params[-1], params[:-1]
    if args_spec.varargs:
        param_args, params = params[-1], params[:-1]
    result = function(*params, *param_args, **param_kwargs)
    if isinstance(result, types.CoroutineType):
        result = await result

    result_payload_key = str(uuid.uuid4())
    async with s3fs.open(result_payload_key, mode='w') as f:
        await f.write(json.dumps(result))
    return result_payload_key


def get_handler(event):
    if "httpMethod" in event:
        return asgi_handler if asgi_handler else init_asgi_handler()
    elif "module_name" in event:
        return rpc_handler
    else:
        raise Exception(f'unsupported invocation. event keys: {list(event.keys())}')


def try_import(module_name):
    from importlib import import_module
    try:
        return import_module(module_name)
    except ModuleNotFoundError as e:
        log.warning(f"{module_name} could not be imported: {e}")


def get_fastapi_app(fastapi, entrypoint):
    fastapi_type = fastapi.FastAPI
    app = getattr(entrypoint, "{{.Expose.ExportedAppVar}}")
    if type(app) is fastapi_type:
        return app
    else:
        log.warning("No FastAPI apps detected.")
