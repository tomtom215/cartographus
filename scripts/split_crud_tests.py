#!/usr/bin/env python3
"""
Split crud_test.go into focused test files by functionality
"""

import re
from pathlib import Path

# Test categorization
CATEGORIES = {
    'insert': ['InsertPlaybackEvent'],
    'upsert': ['UpsertGeolocation'],
    'query': ['SessionKeyExists', 'GetGeolocation', 'GetLastPlaybackTime'],
    'pagination': ['GetPlaybackEvents'],
    'stats': ['GetStats', 'GetUnique']
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

def extract_test_helpers(content: str) -> str:
    """Extract helper functions and types"""
    first_test = re.search(r'^func Test\w+', content, re.MULTILINE)
    if first_test:
        return content[:first_test.start()].rstrip() + '\n\n'
    return ''

def split_tests(content: str) -> dict:
    """Split file content into sections by test function"""
    helpers = extract_test_helpers(content)
    package_match = re.search(r'^package\s+\w+', helpers, re.MULTILINE)
    imports_match = re.search(r'import\s+\([^)]+\)', helpers, re.DOTALL)

    package_line = package_match.group(0) if package_match else 'package database'
    imports_section = imports_match.group(0) if imports_match else ''

    helper_start = imports_match.end() if imports_match else (package_match.end() if package_match else 0)
    helper_funcs = helpers[helper_start:].strip()

    test_pattern = r'(func\s+(Test\w+)\s*\([^)]+\)\s*{)'
    tests = re.finditer(test_pattern, content)

    categorized_tests = {}
    test_positions = []

    for match in tests:
        test_name = match.group(2)
        start_pos = match.start()
        test_positions.append((test_name, start_pos))

    for i, (test_name, start_pos) in enumerate(test_positions):
        if i < len(test_positions) - 1:
            end_pos = test_positions[i + 1][1]
        else:
            end_pos = len(content)

        test_content = content[start_pos:end_pos].rstrip() + '\n'
        category = categorize_test(test_name)

        if category not in categorized_tests:
            categorized_tests[category] = []
        categorized_tests[category].append(test_content)

    return {
        'package': package_line,
        'imports': imports_section,
        'helpers': helper_funcs,
        'tests': categorized_tests
    }

def generate_file_content(package: str, imports: str, helpers: str, tests: list) -> str:
    """Generate content for a split test file"""
    content = f"{package}\n\n"
    content += f"{imports}\n\n"
    if helpers.strip():
        content += f"{helpers}\n\n"
    content += '\n'.join(tests)
    return content

def main():
    source_file = Path('/home/user/map/internal/database/crud_test.go')
    output_dir = Path('/home/user/map/internal/database')

    print(f"Reading {source_file}...")
    content = read_file(source_file)

    print("Splitting tests...")
    sections = split_tests(content)

    print(f"\nFound {len(sections['tests'])} categories:")
    for category, tests in sections['tests'].items():
        print(f"  - {category}: {len(tests)} tests")

    for category, tests in sections['tests'].items():
        output_file = output_dir / f"crud_{category}_split_test.go"
        file_content = generate_file_content(
            sections['package'],
            sections['imports'],
            sections['helpers'],
            tests
        )

        print(f"\nWriting {output_file}...")
        print(f"  Tests: {len(tests)}")
        print(f"  Lines: {len(file_content.splitlines())}")

        with open(output_file, 'w') as f:
            f.write(file_content)

    print("\n✅ Split complete!")
    print(f"Generated {len(sections['tests'])} files")

if __name__ == '__main__':
    main()
