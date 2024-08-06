from typing import Optional, overload, List, Dict

from klotho.construct import ConstructOptions, get_construct_args_opts, Construct
from klotho.output import Input
from klotho.runtime_util import get_default_construct
from klotho.type_util import set_field, get_field

class DynamoDBArgs:
    """Arguments for configuring a DynamoDB table."""

    def __init__(
                 self,
                 attributes: Input[List[Dict[str, str]]],
                 hash_key: Input[str],
                 billing_mode: Optional[Input[str]] = None,
                 range_key: Optional[Input[str]] = None,
                 tags: Optional[Input[Dict[str, str]]] = None
                ):
        set_field(self, "attributes", attributes)
        set_field(self, "hash_key", hash_key)
        if billing_mode is not None:
            set_field(self, "billing_mode", billing_mode)
        if range_key is not None:
            set_field(self, "range_key", range_key)
        if tags is not None:
            set_field(self, "tags", tags)

    def _get_property(self, name: str):
        return get_field(self, name)

    def _set_property(self, name: str, value):
        set_field(self, name, value)

    @property
    def attributes(self) -> Input[List[Dict[str, str]]]:
        return self._get_property("attributes")

    @attributes.setter
    def attributes(self, value: Input[List[Dict[str, str]]]) -> None:
        self._set_property("attributes", value)

    @property
    def billing_mode(self) -> Optional[Input[str]]:
        return self._get_property("billing_mode")

    @billing_mode.setter
    def billing_mode(self, value: Optional[Input[str]]) -> None:
        self._set_property("billing_mode", value)

    @property
    def hash_key(self) -> Input[str]:
        return self._get_property("hash_key")

    @hash_key.setter
    def hash_key(self, value: Input[str]) -> None:
        self._set_property("hash_key", value)

    @property
    def range_key(self) -> Optional[Input[str]]:
        return self._get_property("range_key")

    @range_key.setter
    def range_key(self, value: Optional[Input[str]]) -> None:
        self._set_property("range_key", value)

    @property
    def tags(self) -> Optional[Input[Dict[str, str]]]:
        return self._get_property("tags")

    @tags.setter
    def tags(self, value: Optional[Input[Dict[str, str]]]) -> None:
        self._set_property("tags", value)

class DynamoDB(Construct):
    """Represents a DynamoDB table construct in AWS."""

    @overload
    def __init__(
        self, name: str, args: DynamoDBArgs, opts: Optional[ConstructOptions] = None
    ): ...

    @overload
    def __init__(
        self,
        name: str,
        attributes: Input[List[Dict[str, str]]],
        hash_key: Input[str],
        billing_mode: Optional[Input[str]] = None,
        range_key: Optional[Input[str]] = None,
        tags: Optional[Input[Dict[str, str]]] = None,
        opts: Optional[ConstructOptions] = None,
    ): ...

    def __init__(self, name: str, *args, **kwargs):
        construct_args, opts = get_construct_args_opts(DynamoDBArgs, *args, **kwargs)
        if construct_args is not None:
            self._internal_init(name, opts, **construct_args.__dict__)
        else:
            self._internal_init(name, *args, **kwargs)

    def _internal_init(
        self,
        name: str,
        attributes: Input[List[Dict[str, str]]],
        hash_key: Input[str],
        billing_mode: Optional[Input[str]] = None,
        range_key: Optional[Input[str]] = None,
        tags: Optional[Input[Dict[str, str]]] = None,
        opts: Optional[ConstructOptions] = None,
    ):
        """Internal initializer for DynamoDB."""
        if billing_mode is None:
            billing_mode = "PAY_PER_REQUEST"

        super().__init__(
            name,
            construct_type="klotho.aws.DynamoDB",
            properties={
                "Attributes": attributes,
                "BillingMode": billing_mode,
                "HashKey": hash_key,
                "RangeKey": range_key,
                "Tags": tags,
            },
            opts=opts,
        )

    @property
    def table_name(self) -> str:
        """The name of the DynamoDB table."""
        return get_field(self, "TableName")

    @property
    def table_arn(self) -> str:
        """The Amazon Resource Name (ARN) of the DynamoDB table."""
        return get_field(self, "TableArn")
