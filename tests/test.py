from pathlib import Path
from subprocess import run, PIPE

import pytest


NEW_HEX2RAW = "./hex2raw"
OLD_HEX2RAW = "./hex2raw-old"


def get_test_files(path):
    return sorted(list(path.glob("*.txt")))


@pytest.mark.parametrize("test_file", get_test_files(Path("tests/same")))
def test_same_output(test_file):
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

    # Check if both executables produced the same output
    assert new_result.returncode == old_result.returncode
    assert new_result.stdout == old_result.stdout
    assert new_result.stderr == old_result.stderr


@pytest.mark.parametrize("test_file", get_test_files(Path("tests/different")))
def test_different(test_file):
    # Run the first executable
    new_result = run(
        [NEW_HEX2RAW],
        input=test_file.read_text(),
        stdout=PIPE,
        stderr=PIPE,
        text=True,
    )

    # Run the second executable
    old_result = run(
        [OLD_HEX2RAW],
        input=test_file.read_text(),
        stdout=PIPE,
        stderr=PIPE,
        text=True,
    )

    assert new_result.returncode == 0
    assert old_result.returncode == 255
    assert new_result.stdout == "this is a test\n"
