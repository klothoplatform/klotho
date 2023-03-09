import boto3

secretPrefix = '{{.AppName}}'

def open(path: str, **kwargs):
    return secretsContexManager(path)

class secretsContexManager():
    def __init__(self, path: str):
        self.client = boto3.client('secretsmanager')
        self.secret_name = "{}-{}".format(secretPrefix, path)

    async def __aenter__(self):
        return SecretItem(name=self.secret_name, client=self.client)

    async def __aexit__(self, exc_type, exc_val, traceback):
        pass

class SecretItem():
    def __init__(self, name: str, client: boto3.client, **kwargs):
        self.client = client
        self.name = name

    async def read(self):
        response = self.client.get_secret_value(SecretId=self.name)
        if response.get("SecretBinary"):
            return response["SecretBinary"].decode('utf8')
        elif response.get("SecretString"):
            return response["SecretString"]
        raise Exception("Empty Secret")
