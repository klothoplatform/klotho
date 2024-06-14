import os
import subprocess

def generate_grpc_files(script_dir, proto_path):
    # Define the output directory relative to the script directory
    output_dir = os.path.join(script_dir, 'klothosdk', 'src', 'klotho')
    
    # Run the protoc command using pipenv to ensure the correct environment
    result = subprocess.run([
        'pipenv', 'run', 'python3', '-m', 'grpc_tools.protoc',
        f'-I{os.path.dirname(proto_path)}', 
        f'--python_out={output_dir}', 
        f'--grpc_python_out={output_dir}', 
        os.path.basename(proto_path)
    ], cwd=script_dir, check=True)
    
    print(f"Generated gRPC files in {output_dir}")

def adjust_imports(file_path, old_import, new_import):
    if not os.path.exists(file_path):
        print(f"File {file_path} does not exist.")
        return
    
    with open(file_path, 'r') as file:
        content = file.read()
    
    if old_import in content:
        content = content.replace(old_import, new_import)
        with open(file_path, 'w') as file:
            file.write(content)
        print(f"Updated imports in {file_path}")
    else:
        print(f"No need to update imports in {file_path}")

def main():
    # Get the directory where this script is located
    script_dir = os.path.dirname(os.path.abspath(__file__))
    
    # Define the path to the service.proto file relative to the script directory
    proto_path = os.path.join(script_dir, '..', 'service.proto')
    
    # Define the base path relative to the script directory
    base_path = os.path.join(script_dir, 'klothosdk', 'src', 'klotho')
    service_pb2_grpc_path = os.path.join(base_path, 'service_pb2_grpc.py')
    
    # Generate the gRPC files with paths relative to the script directory
    generate_grpc_files(script_dir, proto_path)
    
    # Adjust the imports in the generated gRPC files
    adjust_imports(service_pb2_grpc_path, 'import service_pb2 as service__pb2', 'import klotho.service_pb2 as service__pb2')

if __name__ == '__main__':
    main()
