from pathlib import Path
from subprocess import run, PIPE

import pytest


NEW_HEX2RAW = "./hex2raw"
OLD_HEX2RAW = "./hex2raw-old"


def get_test_files(path):
    return sorted(list(path.glob("*.txt")))


def run_commands(test_file):
    # Run the first executable
    new_result = run(
        [NEW_HEX2RAW],
        input=test_file.read_bytes(),
        stdout=PIPE,
        stderr=PIPE,
    )

    # Run the second executable
    old_result = run(
        [OLD_HEX2RAW],
        input=test_file.read_bytes(),
        stdout=PIPE,
        stderr=PIPE,
    )

    return new_result, old_result


@pytest.mark.parametrize("test_file", get_test_files(Path("tests/same")))
def test_same_output(test_file):
    new_result, old_result = run_commands(test_file)

    # Check if both executables produced the same output
    assert new_result.returncode == old_result.returncode
    assert new_result.stdout == old_result.stdout
    assert new_result.stderr == old_result.stderr


@pytest.mark.parametrize("test_file", get_test_files(Path("tests/different")))
def test_different(test_file):
    new_result, old_result = run_commands(test_file)

    assert new_result.returncode == 0
    assert old_result.returncode == 255
    assert new_result.stdout == b"this is a test\n"


def test_new_line():
    test_file = Path("tests/test-newline.txt")
    new_result, old_result = run_commands(test_file)

    # They both work and return the same thing...
    assert new_result.returncode == old_result.returncode
    assert new_result.stdout == old_result.stdout

    # ...but the new hex2raw returns something to stderr while the other one doesn't
    assert len(new_result.stderr) > 0
    assert len(old_result.stderr) == 0
