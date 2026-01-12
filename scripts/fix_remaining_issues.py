#!/usr/bin/env python3
"""
Fix remaining go vet issues
"""

import re
from pathlib import Path

def remove_unused_import(file_path: Path, import_name: str) -> bool:
    """Remove an unused import from a file"""
    with open(file_path, 'r') as f:
        content = f.read()

    # Try to remove the import line
    patterns = [
        f'\t"{import_name}"\n',  # Tab-indented with quotes
        f'    "{import_name}"\n',  # Space-indented with quotes
        f'\t{import_name}\n',  # Tab-indented without quotes (for package names)
    ]

    original = content
    for pattern in patterns:
        content = content.replace(pattern, '')

    if content != original:
        with open(file_path, 'w') as f:
            f.write(content)
        return True
    return False

def find_and_move_helper_to_shared(file_pattern: str, helper_name: str, shared_file: Path) -> None:
    """Find helper function in split files and move to shared file"""
    files = list(Path('/home/user/map/internal/sync').glob(file_pattern))

    # Find the helper in the first file
    helper_code = None
    for file_path in files:
        with open(file_path, 'r') as f:
            content = f.read()

        # Match helper function
        match = re.search(
            rf'(// [^\n]*\n)?func {helper_name}\([^{{]+\{{[^}}]+\}})',
            content,
            re.DOTALL
        )

        if match and helper_code is None:
            helper_code = match.group(0)
            print(f"  Found {helper_name} in {file_path.name}")

    # Add to shared file if found
    if helper_code:
        with open(shared_file, 'a') as f:
            f.write(f'\n{helper_code}\n')
        print(f"  ✓ Added {helper_name} to {shared_file.name}")

    # Remove from all files
    for file_path in files:
        with open(file_path, 'r') as f:
            content = f.read()

        # Remove helper function
        new_content = re.sub(
            rf'(// [^\n]*\n)?func {helper_name}\([^{{]+\{{[^}}]+\}}\n*',
            '',
            content,
            flags=re.DOTALL
        )

        if new_content != content:
            with open(file_path, 'w') as f:
                f.write(new_content)
            print(f"  ✓ Removed {helper_name} from {file_path.name}")

def main():
    print("Fixing remaining go vet issues...\n")

    # Issue 1: Remove "fmt" from crud_query_split_test.go
    print("1. Removing unused 'fmt' import from crud_query_split_test.go...")
    file1 = Path('/home/user/map/internal/database/crud_query_split_test.go')
    if remove_unused_import(file1, 'fmt'):
        print("  ✓ Removed 'fmt' import\n")
    else:
        print("  ! Could not remove 'fmt' import\n")

    # Issue 2: Move stringPtr to test_helpers.go and remove from split files
    print("2. Moving stringPtr helper to test_helpers.go...")
    shared_file = Path('/home/user/map/internal/sync/test_helpers.go')
    find_and_move_helper_to_shared('tautulli_*_split_test.go', 'stringPtr', shared_file)
    print()

    # Issue 3: Remove unused imports from handlers_filters_split_test.go
    print("3. Removing unused imports from handlers_filters_split_test.go...")
    file3 = Path('/home/user/map/internal/api/handlers_filters_split_test.go')
    removed = []
    for import_name in ['github.com/tomtom215/map/internal/auth',
                        'github.com/tomtom215/map/internal/cache',
                        'github.com/tomtom215/map/internal/config',
                        'github.com/tomtom215/map/internal/models',
                        'github.com/tomtom215/map/internal/models/tautulli']:
        if remove_unused_import(file3, import_name):
            removed.append(import_name.split('/')[-1])

    if removed:
        print(f"  ✓ Removed: {', '.join(removed)}\n")
    else:
        print("  - No changes needed\n")

    print("✅ Done!")

if __name__ == '__main__':
    main()
