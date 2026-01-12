#!/usr/bin/env python3
"""
Generate stub implementations for all TautulliClientInterface methods
"""

import re
from pathlib import Path

def extract_interface_methods(interface_file: Path) -> list:
    """Extract method signatures from TautulliClientInterface"""
    with open(interface_file, 'r') as f:
        content = f.read()

    # Find the interface definition
    interface_match = re.search(
        r'type TautulliClientInterface interface \{([^}]+)\}',
        content,
        re.DOTALL
    )

    if not interface_match:
        return []

    interface_body = interface_match.group(1)

    # Extract method signatures (ignore comments)
    methods = []
    for line in interface_body.split('\n'):
        line = line.strip()
        if line and not line.startswith('//'):
            methods.append(line)

    return methods

def generate_stub_method(signature: str) -> str:
    """Generate a stub implementation for a method signature"""
    # Parse method signature
    match = re.match(r'(\w+)\([^)]*\)\s*\(([^)]+)\)', signature)
    if not match:
        return ""

    method_name = match.group(1)
    return_type = match.group(2).strip()

    # Extract the actual return type (remove error)
    if ', error' in return_type:
        return_type = return_type.replace(', error', '').strip()

    # Generate stub
    stub = f"""
func (m *mockTautulliClient) {signature.strip()} {{
	return {return_type}{{}}, nil
}}
"""
    return stub

def main():
    interface_file = Path('/home/user/map/internal/sync/tautulli_client.go')

    methods = extract_interface_methods(interface_file)

    print(f"Found {len(methods)} methods in TautulliClientInterface")
    print("\nGenerate stub methods for mockTautulliClient:")
    print("=" * 80)

    # Methods already implemented
    implemented = ['GetHistorySince', 'GetGeoIPLookup', 'Ping', 'DeleteExport']

    for method in methods:
        method_name = method.split('(')[0]
        if method_name not in implemented:
            stub = generate_stub_method(method)
            if stub:
                print(stub)

if __name__ == '__main__':
    main()
