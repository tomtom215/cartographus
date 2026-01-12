#!/usr/bin/env python3
"""
Split tautulli_test.go into focused test files by functionality
"""

import re
from pathlib import Path
from typing import List, Tuple

# Test categorization based on function names
CATEGORIES = {
    'history': ['GetHistory', 'GetHistorySince', '_Errors'],
    'geoip': ['GeoIPLookup', 'GeoIP'],
    'activity': ['GetActivity', 'GetMetadata', 'GetStreamData'],
    'users': ['GetUser', 'UserPlayerStats', 'UserWatchTimeStats', 'UserIPs', 'UsersTable', 'UserLogins'],
    'libraries': ['GetLibraries', 'GetLibrary', 'LibraryUserStats', 'LibrariesTable', 'LibraryMediaInfo',
                  'LibraryWatchTimeStats', 'GetRecentlyAdded', 'GetChildrenMetadata', 'LibraryNames'],
    'server': ['GetServerInfo', 'GetSyncedItems', 'TerminateSession'],
    'analytics': ['GetHomeStats', 'GetPlaysByDate', 'GetPlaysByDayOfWeek', 'GetPlaysByHourOfDay',
                  'GetPlaysByStreamType', 'ConcurrentStreams', 'ItemWatchTimeStats', 'PlaysBySourceResolution',
                  'PlaysByStreamResolution', 'PlaysByTop10', 'PlaysPerMonth', 'ItemUserStats'],
    'export': ['ExportMetadata', 'GetExportFields'],
    'client_basic': ['NewTautulliClient', '_Ping', 'NetworkFailure', 'Timeout']
}

def categorize_test(test_name: str) -> str:
    """Determine which category a test belongs to"""
    for category, keywords in CATEGORIES.items():
        for keyword in keywords:
            if keyword in test_name:
                return category
    return 'misc'

def read_file(filepath: str) -> str:
    """Read file content"""
    with open(filepath, 'r') as f:
        return f.read()

def split_tests(content: str) -> dict:
    """Split file content into sections by test function"""
    # Extract package and imports
    package_match = re.search(r'^package\s+\w+', content, re.MULTILINE)
    imports_match = re.search(r'import\s+\([^)]+\)', content, re.DOTALL)

    package_line = package_match.group(0) if package_match else 'package sync'
    imports_section = imports_match.group(0) if imports_match else ''

    # Find all test functions and their content
    test_pattern = r'(func\s+(Test\w+)\s*\([^)]+\)\s*{)'
    tests = re.finditer(test_pattern, content)

    categorized_tests = {}
    test_positions = []

    for match in tests:
        test_name = match.group(2)
        start_pos = match.start()
        test_positions.append((test_name, start_pos))

    # Extract content for each test
    for i, (test_name, start_pos) in enumerate(test_positions):
        # Find end position (start of next test or end of file)
        if i < len(test_positions) - 1:
            end_pos = test_positions[i + 1][1]
        else:
            # Last test - find helper functions or EOF
            helper_match = re.search(r'\n// Helper functions', content[start_pos:])
            if helper_match:
                end_pos = start_pos + helper_match.start()
            else:
                end_pos = len(content)

        test_content = content[start_pos:end_pos].rstrip() + '\n'
        category = categorize_test(test_name)

        if category not in categorized_tests:
            categorized_tests[category] = []
        categorized_tests[category].append(test_content)

    # Extract helper functions
    helper_match = re.search(r'// Helper functions.*$', content, re.DOTALL)
    helper_functions = helper_match.group(0) if helper_match else ''

    return {
        'package': package_line,
        'imports': imports_section,
        'tests': categorized_tests,
        'helpers': helper_functions
    }

def generate_file_content(package: str, imports: str, tests: List[str], helpers: str = '') -> str:
    """Generate content for a split test file"""
    content = f"{package}\n\n"
    content += f"{imports}\n\n"
    content += '\n'.join(tests)
    if helpers and 'stringPtr' in '\n'.join(tests):
        content += f"\n{helpers}"
    return content

def main():
    source_file = Path('/home/user/map/internal/sync/tautulli_test.go')
    output_dir = Path('/home/user/map/internal/sync')

    print(f"Reading {source_file}...")
    content = read_file(source_file)

    print("Splitting tests...")
    sections = split_tests(content)

    print(f"\nFound {len(sections['tests'])} categories:")
    for category, tests in sections['tests'].items():
        print(f"  - {category}: {len(tests)} tests")

    # Generate output files
    for category, tests in sections['tests'].items():
        output_file = output_dir / f"tautulli_{category}_split_test.go"
        file_content = generate_file_content(
            sections['package'],
            sections['imports'],
            tests,
            sections['helpers']
        )

        print(f"\nWriting {output_file}...")
        print(f"  Tests: {len(tests)}")
        print(f"  Lines: {len(file_content.splitlines())}")

        with open(output_file, 'w') as f:
            f.write(file_content)

    print("\n✅ Split complete!")
    print(f"Generated {len(sections['tests'])} files")
    print("\nNext steps:")
    print("1. Run: go test ./internal/sync/... -v")
    print("2. Verify all tests pass")
    print("3. Delete original tautulli_test.go if tests pass")

if __name__ == '__main__':
    main()
