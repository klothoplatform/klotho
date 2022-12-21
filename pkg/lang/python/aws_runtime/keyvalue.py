import logging
import sys
from datetime import datetime, timedelta
from typing import Optional

import boto3
from aiocache.base import BaseCache
from aiocache.serializers import BaseSerializer
from boto3.dynamodb.conditions import Key, Attr, And, Or
from cerealbox.dynamo import from_dynamodb_json, as_dynamodb_json

log = logging.getLogger("klotho")


class KVItem:

    def __init__(self, pk: str, sk: str, value, expiration: int = None, last_modified_ts: int = None):
        self.pk = pk
        self.sk = sk
        self.value = value
        self.expiration = expiration
        self.last_modified_ts = last_modified_ts


class KVStore(BaseCache):

    def __init__(self, client: boto3.client = None, resource: boto3.resource = None, map_id: str = None,
                 table_name: str = None, **kwargs):
        super().__init__(**kwargs)
        self.dynamodb = boto3.resource('dynamodb') if resource is None else resource
        self.client = boto3.client('dynamodb') if client is None else client
        self.table_name = table_name
        self.map_id = map_id
        self.table = self.dynamodb.Table(self.table_name)

    def _new_item(self, key, value, ttl_seconds=None):
        pk = self._pk()
        sk = self._sk(key)
        return KVItem(pk, sk, value, _ttl_epoch(ttl_seconds), int(datetime.now().timestamp()))

    def _pk(self):
        return self.map_id

    def _sk(self, key):
        return key

    async def _add(self, key, value, ttl, _conn=None):
        item = self._new_item(key, value, ttl)
        return self.table.put_item(
            Item=vars(item),
            ConditionExpression=And(Attr("pk").not_exists(), Attr("sk").not_exists())
        )

    async def _get(self, key, encoding, _conn=None):
        now = int(datetime.now().timestamp())
        result = self.table.get_item(Key={"pk": self._pk(), "sk": self._sk(key)})
        item = result.get("Item", None)
        item = item if item is not None and (item.get("expiration", None) or sys.maxsize) > now else None
        return item.get("value", None) if item else None

    async def _multi_get(self, keys, encoding, _conn=None):
        now = int(datetime.now().timestamp())

        chunks = _chunk_list(keys)
        log.debug(f"_multi_get: keys split into {len(chunks)} chunks")
        results = [self._get_batch(c) for c in chunks]

        return [item.get("value", None)
                for chunk in results
                for item in chunk
                if item is not None and (item.get("expiration", None) or sys.maxsize) > now]

    def _get_batch(self, keys) -> list[Optional[dict]]:
        sorted_responses = [None] * len(keys)
        item_order = {key: index for index, key in enumerate(keys)}
        log.debug(f"_get_batch: getting {len(keys)} item(s)")

        query_keys = [{"pk": self._pk(), "sk": self._sk(key)} for key in keys]
        # TODO: look into adding a more comprehensive retry strategy for unprocessed keys
        for a in range(0, 3):
            log.debug(f"_get_batch: attempt {a + 1}")
            result = self.dynamodb.batch_get_item(
                RequestItems={
                    self.table_name: {
                        "Keys": query_keys
                    }})
            count = 0
            for i in result.get("Responses", {}).get(self.table_name, []):
                sorted_responses[item_order[i["sk"]]] = i
                count += 1
            log.debug(f"_get_batch: got {count} items")
            unprocessed = result.get("UnprocessedKeys", {}).get(self.table_name, {}).get("Keys", [])
            log.debug(f"_get_batch: {len(unprocessed)} unprocessed key(s) in batch")
            if len(unprocessed) == 0:
                return sorted_responses
            query_keys = unprocessed

        raise Exception("some items could not be processed by DynamoDB")

    async def _set(self, key, value, ttl, _cas_token=None, _conn=None):
        item = self._new_item(key, value, ttl)
        self.table.put_item(Item=vars(item))

    async def _multi_set(self, pairs, ttl, _conn=None):
        with self.table.batch_writer() as batch:
            for key, value in pairs:
                item = self._new_item(key, value, ttl)
                batch.put_item(Item=vars(item))

    async def _delete(self, key, _conn=None):
        self.table.delete_item(Key={"pk": self._pk(), "sk": self._sk(key)})

    async def _exists(self, key, _conn=None):
        result = self.table.get_item(
            Key={"pk": self._pk(), "sk": self._sk(key)},
            AttributesToGet=["pk"])
        return result.get("Item", None) is not None

    async def _increment(self, key, delta, _conn=None):
        raise NotImplementedError("_increment is not implemented")

    async def _expire(self, key, ttl, _conn=None):
        self.table.update_item(
            Key={"pk": self._pk(), "sk": self._sk(key)},
            AttributeUpdates={"expiration": {"Value": _ttl_epoch(ttl), "Action": "PUT"}})

    async def _clear(self, namespace, _conn=None):
        now = int(datetime.now().timestamp())
        key_expr = Key("pk").eq(self._pk())
        filter_expr = Or(Attr("last_modified_ts").lte(now), Attr("last_modified_ts").not_exists())

        result = self.table.query(
            KeyConditionExpression=key_expr,
            FilterExpression=filter_expr,
        )

        count = 0
        with self.table.batch_writer() as batch:
            while True:
                kv_items = result.get("Items", [])
                last_evaluated_key = result.get("LastEvaluatedKey", None)

                if len(kv_items) == 0:
                    break

                for item in kv_items:
                    batch.delete_item(Key={"pk": item["pk"], "sk": item["sk"]})
                    count += 1

                if last_evaluated_key is None:
                    break

                result = self.table.query(
                    KeyConditionExpression=key_expr,
                    FilterExpression=filter_expr,
                    ExclusiveStartKey=last_evaluated_key,
                )
        return count

    async def _raw(self, command, *args, **kwargs):
        raise NotImplementedError("_raw is not implemented")


class DynamoDBSerializer(BaseSerializer):
    def dumps(self, value):
        val = from_dynamodb_json(as_dynamodb_json(value))
        return val

    def loads(self, value):
        return value


def _ttl_epoch(ttl_seconds: int):
    return (None if ttl_seconds is None or ttl_seconds == 0
            else int((datetime.now() + timedelta(seconds=ttl_seconds)).timestamp()))


def _chunk_list(items, max_items=100):
    return [items[i:i + max_items] for i in range(0, len(items), max_items)]
