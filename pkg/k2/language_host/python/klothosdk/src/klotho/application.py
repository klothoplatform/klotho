import os

from klotho.runtime import instance as runtime


class Application:
    def __init__(self, name, project=None, environment=None, default_region=None):
        self.name = name
        self.project = project or os.getenv('PROJECT_NAME', 'default')
        self.environment = environment or os.getenv('KLOTHO_ENVIRONMENT', 'default')
        self.default_region = default_region or os.getenv('AWS_REGION', 'us-east-1')
        runtime.set_application(self)
