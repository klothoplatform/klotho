import re


class URN:
    def __init__(self, account_id="", project="", environment="", application="", type="", subtype="",
                 parent_resource_id="", resource_id="", output=""):
        self.account_id = account_id
        self.project = project
        self.environment = environment
        self.application = application
        self.type = type
        self.subtype = subtype
        self.parent_resource_id = parent_resource_id
        self.resource_id = resource_id
        self.output = output

    @classmethod
    def parse(cls, urn_string):
        matches = re.findall('[^:]+', urn_string)

        if len(matches) < 2:
            raise ValueError("Invalid URN format")

        if matches[0] == "urn":
            matches = matches[1:]

        urn = cls(account_id=matches[0], project=matches[1])

        if len(matches) > 2 and matches[2] != "":
            urn.environment = matches[2]
        if len(matches) > 3 and matches[3] != "":
            urn.application = matches[3]
        if len(matches) > 4 and matches[4] != "":
            type_parts = matches[4].split("/")
            if len(type_parts) != 2:
                raise ValueError("Invalid URN type format")
            urn.type = type_parts[0]
            urn.subtype = type_parts[1]

        if len(matches) > 5 and matches[5] != "":
            resource_parts = matches[5].split("/")
            if len(resource_parts) == 2:
                urn.parent_resource_id = resource_parts[0]
                urn.resource_id = resource_parts[1]
            else:
                urn.resource_id = matches[5]
        if len(matches) > 6 and matches[6] != "":
            urn.output = matches[6]

        if len(matches) > 7:
            raise ValueError("Invalid URN format")

        return urn

    def __str__(self):
        urn = f"urn:{self.account_id}:{self.project}:{self.environment}:{self.application}:"
        if self.type and self.subtype:
            urn += f"{self.type}/{self.subtype}:"
        if self.parent_resource_id and self.resource_id:
            urn += f"{self.parent_resource_id}/{self.resource_id}:"
        else:
            urn += f"{self.resource_id}:"
        urn += f"{self.output}:"

        # Remove trailing colons
        return urn.rstrip(":")

    def clone(self):
        return URN(
            account_id=self.account_id,
            project=self.project,
            environment=self.environment,
            application=self.application,
            type=self.type,
            subtype=self.subtype,
            parent_resource_id=self.parent_resource_id,
            resource_id=self.resource_id,
            output=self.output
        )

    def with_output(self, output: str):
        return URN(
            account_id=self.account_id,
            project=self.project,
            environment=self.environment,
            application=self.application,
            type=self.type,
            subtype=self.subtype,
            parent_resource_id=self.parent_resource_id,
            resource_id=self.resource_id,
            output=output
        )

    def __eq__(self, other):
        return str(self) == str(other)

    def __hash__(self):
        return hash(str(self))
