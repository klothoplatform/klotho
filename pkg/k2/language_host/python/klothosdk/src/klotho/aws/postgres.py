from typing import Optional, overload

from klotho.aws.network import Network
from klotho.construct import ConstructOptions, get_construct_args_opts, Construct
from klotho.output import Input
from klotho.runtime_util import get_default_construct
from klotho.type_util import set_field, get_field


class PostgresArgs:
    """Arguments for configuring a Postgres database."""

    def __init__(
                 self,
                 database_name: Input[str],
                 instance_class: Optional[Input[str]] = None,
                 allocated_storage: Optional[Input[int]] = None,
                 engine_version: Optional[Input[str]] = None,
                 username: Optional[Input[str]] = None,
                 password: Optional[Input[str]] = None,
                 port: Optional[Input[int]] = None,
                 network: Optional[Network] = None
                ):
        if instance_class is not None:
            set_field(self, "instance_class", instance_class)
        if allocated_storage is not None:
            set_field(self, "allocated_storage", allocated_storage)
        if engine_version is not None:
            set_field(self, "engine_version", engine_version)
        if username is not None:
            set_field(self, "username", username)
        if password is not None:
            set_field(self, "password", password)
        if database_name is not None:
            set_field(self, "database_name", database_name)
        if port is not None:
            set_field(self, "port", port)
        if network is not None:
            set_field(self, "network", network)

    def _get_property(self, name: str):
        return get_field(self, name)

    def _set_property(self, name: str, value):
        set_field(self, name, value)

    @property
    def instance_class(self) -> Optional[Input[str]]:
        return self._get_property("instance_class")

    @instance_class.setter
    def instance_class(self, value: Optional[Input[str]]) -> None:
        self._set_property("instance_class", value)

    @property
    def allocated_storage(self) -> Optional[Input[int]]:
        return self._get_property("allocated_storage")

    @allocated_storage.setter
    def allocated_storage(self, value: Optional[Input[int]]) -> None:
        self._set_property("allocated_storage", value)

    @property
    def engine_version(self) -> Optional[Input[str]]:
        return self._get_property("engine_version")

    @engine_version.setter
    def engine_version(self, value: Optional[Input[str]]) -> None:
        self._set_property("engine_version", value)

    @property
    def username(self) -> Optional[Input[str]]:
        return self._get_property("username")

    @username.setter
    def username(self, value: Optional[Input[str]]) -> None:
        self._set_property("username", value)

    @property
    def password(self) -> Optional[Input[str]]:
        return self._get_property("password")

    @password.setter
    def password(self, value: Optional[Input[str]]) -> None:
        self._set_property("password", value)

    @property
    def database_name(self) -> Optional[Input[str]]:
        return self._get_property("database_name")

    @database_name.setter
    def database_name(self, value: Optional[Input[str]]) -> None:
        self._set_property("database_name", value)

    @property
    def port(self) -> Optional[Input[int]]:
        return self._get_property("port")

    @port.setter
    def port(self, value: Optional[Input[int]]) -> None:
        self._set_property("port", value)

    @property
    def network(self) -> Network:
        return self._get_property("network")

    @network.setter
    def network(self, value: Optional[Network]) -> None:
        self._set_property("network", value)


class Postgres(Construct):
    """Represents a Postgres database construct in AWS."""

    @overload
    def __init__(
        self, name: str, args: PostgresArgs, opts: Optional[ConstructOptions] = None
    ): ...

    @overload
    def __init__(
        self,
        name: str,
        instance_class: Optional[Input[str]] = None,
        allocated_storage: Optional[Input[int]] = None,
        engine_version: Optional[Input[str]] = None,
        username: Optional[Input[str]] = None,
        password: Optional[Input[str]] = None,
        database_name: Optional[Input[str]] = None,
        port: Optional[Input[int]] = None,
        network: Optional[Network] = None,
        opts: Optional[ConstructOptions] = None,
    ): ...

    def __init__(self, name: str, *args, **kwargs):
        construct_args, opts = get_construct_args_opts(PostgresArgs, *args, **kwargs)
        if construct_args is not None:
            self._internal_init(name, opts, **construct_args.__dict__)
        else:
            self._internal_init(name, *args, **kwargs)

    def _internal_init(
        self,
        name: str,
        instance_class: Optional[Input[str]] = None,
        allocated_storage: Optional[Input[int]] = None,
        opts: Optional[ConstructOptions] = None,
        engine_version: Optional[Input[str]] = None,
        username: Optional[Input[str]] = None,
        password: Optional[Input[str]] = None,
        database_name: Optional[Input[str]] = None,
        port: Optional[Input[int]] = None,
        network: Optional[Network] = None,
    ):
        """Internal initializer for Postgres."""
        if network is None:
            network = get_default_construct("aws", Network)

        if instance_class is None:
            instance_class = "db.t3.micro"
        if allocated_storage is None:
            allocated_storage = 20

        super().__init__(
            name,
            construct_type="klotho.aws.Postgres",
            properties={
                "InstanceClass": instance_class,
                "AllocatedStorage": allocated_storage,
                "EngineVersion": engine_version,
                "Username": username,
                "Password": password,
                "DatabaseName": database_name,
                "Port": port,
                "Network": network,
            },
            opts=opts,
        )

    @property
    def endpoint(self) -> str:
        """The endpoint of the Postgres database."""
        return get_field(self, "Endpoint")

    @property
    def port(self) -> int:
        """The port of the Postgres database."""
        return get_field(self, "Port")
