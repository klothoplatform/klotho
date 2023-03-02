import json
import os
import yaml
from pathlib import Path


def get_pulumi_resources():
    if Path('test_resources.json').exists():
        with open('test_resources.json') as output:
            output_json = json.loads(output)
            resources = output_json.get('deployment', {}).get('resources', [])
            return {
                "klo:vpc": {
                    "id": next(r for r in resources if r["type"] == "aws:ec2/vpc:Vpc")['id'],
                    "sgId": next(r for r in resources if r["type"] == "aws:ec2/securityGroup:SecurityGroup")['id'],
                    "privateSubnetIds": [
                        r['id'] for r in resources if r['type'] == 'aws:ec2/subnet:Subnet' and 'private' in r['urn'].split('::')[-1]
                    ],
                    "publicSubnetIds": [
                        r['id'] for r in resources if r['type'] == 'aws:ec2/subnet:Subnet' and 'public' in r['urn'].split('::')[-1]
                    ],
                }
            }
    else:
        return None

def update_pulumi_app_config(app_yaml_path, resources):
    with open(app_yaml_path) as f:
        app_config = yaml.safe_load(f)

    app_config['config']['aws:region'] = os.environ.get("AWS_REGION", 'us-east-2')
    if resources is not None:
        for k, v in resources.items():
            app_config['config'][k] = v

    with open(app_yaml_path, 'w') as f:
        yaml.dump(app_config, f)

    print(f"Added: {resources} to {app_yaml_path}")

if __name__ == "__main__":
    import sys
    update_pulumi_app_config(sys.argv[1], get_pulumi_resources())
