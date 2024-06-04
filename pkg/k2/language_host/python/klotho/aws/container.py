from ..resource import Resource

class Container(Resource):
    def __init__(self, name, image, source_hash=None, cpu=256, memory=512, context=None, dockerfile=None, port=None, network=None):
        super().__init__(name, "klotho.aws.Container")
        self.add_input("SourceHash", "string", source_hash)
        self.add_input("Cpu", "number", cpu)
        self.add_input("Memory", "number", memory)
        self.add_input("Context", "string", context)
        self.add_input("Dockerfile", "string", dockerfile)
        self.add_input("Image", "string", image)
        self.add_input("Port", "number", port)
        self.add_input("Network", "Construct<klotho.aws.Network>", network)
