#!/usr/bin/env python3
"""
Fix manager split test files by removing all code between imports and first test function
"""

import re
from pathlib import Path

def fix_file(file_path: Path) -> bool:
    """Remove everything between imports and first test function"""
    with open(file_path, 'r') as f:
        content = f.read()

    # Find the end of imports (closing parenthesis of import block)
    imports_match = re.search(r'import\s+\([^)]+\)', content, re.DOTALL)
    if not imports_match:
        print(f"  ! Could not find imports in {file_path.name}")
        return False

    imports_end = imports_match.end()

    # Find the first test function
    first_test_match = re.search(r'^func Test\w+', content, re.MULTILINE)
    if not first_test_match:
        print(f"  ! Could not find test function in {file_path.name}")
        return False

    first_test_start = first_test_match.start()

    # Build new content: imports + newline + tests
    new_content = content[:imports_end] + '\n\n' + content[first_test_start:]

    # Write back
    with open(file_path, 'w') as f:
        f.write(new_content)

    return True

def main():
    sync_dir = Path('/home/user/map/internal/sync')
    manager_files = list(sync_dir.glob('manager_*_split_test.go'))

    print(f"Fixing {len(manager_files)} manager split test files...")

    for file_path in manager_files:
        print(f"Processing {file_path.name}...")
        if fix_file(file_path):
            print(f"  ✓ Fixed")
        else:
            print(f"  ✗ Failed")

    print("\n✅ Done!")

if __name__ == '__main__':
    main()
