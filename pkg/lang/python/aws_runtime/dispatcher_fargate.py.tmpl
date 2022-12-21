import logging
import os
import multiprocessing
import types

app_port = os.getenv("KLOTHO_APP_PORT", 3000)
log_level = os.getenv("KLOTHO_LOG_LEVEL", "DEBUG").upper()
uvicorn_log_level = os.getenv("UVICORN_LOG_LEVEL", "info").lower()
host = "0.0.0.0"

logging.basicConfig()
log = logging.getLogger("klotho")
log.setLevel(logging.DEBUG)


def userland_main():
    main_module_name = "{{.Expose.AppModule}}"
    if not main_module_name:
        log.info("No main defined. Will only listen as a proxy server for RPC calls.")
        return
    entrypoint = try_import(main_module_name)

    if entrypoint is None:
        log.error("Startup failed: No entrypoint found!")
        exit(1)

    uvicorn = try_import("uvicorn")
    fastapi = try_import("fastapi")

    run_fastapi_app = None
    if uvicorn is not None and fastapi is not None:
        run_fastapi_app = run_fastapi_app_func(uvicorn=uvicorn, fastapi=fastapi, entrypoint=entrypoint)

    if run_fastapi_app is not None:
        log.debug("Starting FastAPI app...")
        run_fastapi_app()
    else:
        try:
            entrypoint.__klotho_main__()
        except AttributeError as err:
            log.error(err)


def start_proxy_server():
    import fastapi
    import uvicorn
    klotho_proxy_app = fastapi.FastAPI()

    @klotho_proxy_app.get("/")
    async def proxy_root_get():
        return

    @klotho_proxy_app.post("/")
    async def proxy_root_post(obj: dict):
        module_name, function_name, params = obj['module_name'], obj['function_to_call'], obj['params']
        module_obj = try_import(module_name)
        if not module_obj:
            raise Exception(f"couldn't find module: {module_name}")
        function = getattr(module_obj, function_name, None)
        if not function:
            raise Exception(f"couldn't find function: {module_name}.{function_name}")
        result = function(*params)
        if isinstance(result, types.CoroutineType):
            result = await result
        return result

    uvicorn.run(
        klotho_proxy_app,
        host=host,
        port=3001,
        log_level=uvicorn_log_level)


def try_import(module_name):
    from importlib import import_module
    try:
        return import_module(module_name)
    except ModuleNotFoundError as e:
        log.debug(f"{module_name} could not be imported: {e}")


def run_fastapi_app_func(uvicorn, fastapi, entrypoint):
    fastapi_type = fastapi.FastAPI
    api = getattr(entrypoint, "{{.Expose.ExportedAppVar}}")
    if type(api) is fastapi_type:
        def func():
            uvicorn.run(
                f"{entrypoint.__name__}:{{.Expose.ExportedAppVar}}",
                host=host,
                port=app_port,
                log_level=uvicorn_log_level)

        return func
    else:
        log.debug("No FastAPI apps detected.")


if __name__ == "__main__":
    subprocesses = [multiprocessing.Process(target=m) for m in [userland_main, start_proxy_server]]
    for sp in subprocesses:
        sp.start()
    for sp in subprocesses:
        sp.join()
