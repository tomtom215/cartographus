#!/usr/bin/env python3
"""
Remove duplicate mockDB and mockTautulliClient declarations from split test files
"""

import re
from pathlib import Path

def remove_mock_declarations(file_path: Path) -> bool:
    """Remove mockDB and mockTautulliClient type declarations from file"""
    with open(file_path, 'r') as f:
        content = f.read()

    # Pattern to match the mock type declarations and their methods
    # Match from "// Mock database" to the end of mockDB methods
    mockdb_pattern = r'// Mock database for testing\ntype mockDB struct \{[^}]+\}\n\n(?:func \(m \*mockDB\) \w+\([^)]*\)[^{]*\{[^}]+\}\n\n?)+'

    # Match from "// Mock Tautulli" to the end of mockTautulliClient methods
    mockclient_pattern = r'// Mock Tautulli client for testing\ntype mockTautulliClient struct \{[^}]+\}\n\n(?:func \(m \*mockTautulliClient\) \w+\([^)]*\)[^{]*\{(?:[^}]|\}(?!\n\n))+\}\n\n?)+'

    original_content = content

    # Remove mock declarations
    content = re.sub(mockdb_pattern, '', content, flags=re.DOTALL)
    content = re.sub(mockclient_pattern, '', content, flags=re.DOTALL)

    # If content changed, write it back
    if content != original_content:
        with open(file_path, 'w') as f:
            f.write(content)
        return True
    return False

def main():
    sync_dir = Path('/home/user/map/internal/sync')

    # Find all manager split test files
    manager_files = list(sync_dir.glob('manager_*_split_test.go'))

    print(f"Found {len(manager_files)} manager split test files")

    for file_path in manager_files:
        print(f"Processing {file_path.name}...")
        if remove_mock_declarations(file_path):
            print(f"  ✓ Removed duplicate mock declarations")
        else:
            print(f"  - No changes needed")

    print("\n✅ Done!")

if __name__ == '__main__':
    main()
