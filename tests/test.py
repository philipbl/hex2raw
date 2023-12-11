from pathlib import Path
import secrets
from subprocess import run, PIPE

import pytest


NEW_HEX2RAW = "./hex2raw"
OLD_HEX2RAW = "./hex2raw-old"


def get_test_files(path):
    return sorted(list(path.glob("*.txt")))


def get_test_data(num_tests, len_test):
    for i in range(num_tests):
        sequence_size = secrets.randbelow(len_test) + 1
        hex_sequence = (secrets.token_hex(1).encode() for _ in range(sequence_size))

        # Remove all "0a"
        hex_sequence = (x for x in hex_sequence if x != b"0a")

        yield b" ".join(hex_sequence)


def run_commands(data):
    # Run the first executable
    new_result = run(
        [NEW_HEX2RAW],
        input=data,
        stdout=PIPE,
        stderr=PIPE,
    )

    # Run the second executable
    old_result = run(
        [OLD_HEX2RAW],
        input=data,
        stdout=PIPE,
        stderr=PIPE,
    )

    return new_result, old_result


@pytest.mark.parametrize("test_file", get_test_files(Path("tests/same")))
def test_same_output(test_file):
    new_result, old_result = run_commands(test_file.read_bytes())

    # Check if both executables produced the same output
    assert new_result.returncode == old_result.returncode
    assert new_result.stdout == old_result.stdout
    assert new_result.stderr == old_result.stderr


@pytest.mark.parametrize("test_file", get_test_files(Path("tests/different")))
def test_different(test_file):
    new_result, old_result = run_commands(test_file.read_bytes())

    assert new_result.returncode == 0
    assert old_result.returncode == 255
    assert new_result.stdout == b"this is a test\n"


def test_new_line():
    test_file = Path("tests/test-newline.txt")
    new_result, old_result = run_commands(test_file.read_bytes())

    # They both work and return the same thing...
    assert new_result.returncode == old_result.returncode
    assert new_result.stdout == old_result.stdout

    # ...but the new hex2raw returns something to stderr while the other one doesn't
    assert len(new_result.stderr) > 0
    assert len(old_result.stderr) == 0


@pytest.mark.parametrize("test_data", get_test_data(100, 100))
def test_random(test_data):
    new_result, old_result = run_commands(test_data)

    # Check if both executables produced the same output
    assert new_result.returncode == old_result.returncode
    assert new_result.stdout == old_result.stdout
    assert new_result.stderr == old_result.stderr
