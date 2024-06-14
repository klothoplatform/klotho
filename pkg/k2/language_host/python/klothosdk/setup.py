from setuptools import setup, find_packages
import os

# Read the contents of the README file
with open("README.md", "r", encoding="utf-8") as fh:
    long_description = fh.read()


setup(
    name="klotho",
    version="0.1.0",
    author="Aaron Torres",
    author_email="atorres@klo.dev",
    description="A Python SDK for Klotho2",
    long_description=long_description,
    long_description_content_type="text/markdown",
    packages=find_packages(where="src"),
    package_dir={"": "src"},
    classifiers=[
        "Programming Language :: Python :: 3",
        "License :: OSI Approved :: MIT License",
        "Operating System :: OS Independent",
    ],
    python_requires=">=3.6",
    install_requires=[
        "argparse==1.4.0",
        "grpcio==1.64.0; python_version >= '3.8'",
        "grpcio-tools==1.64.0; python_version >= '3.8'",
        "protobuf==5.27.0; python_version >= '3.8'",
        "pyyaml==6.0.1; python_version >= '3.6'",
        "setuptools==70.0.0; python_version >= '3.8'"
    ],
    extras_require={
        "dev": [
            "debugpy==1.8.1; python_version >= '3.8'"
        ]
    },
    include_package_data=True,
)
