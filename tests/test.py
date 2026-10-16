from pathlib import Path
import random
from subprocess import run, PIPE

import pytest


NEW_HEX2RAW = "./hex2raw"
OLD_HEX2RAW = "./hex2raw-old"

# A hang should fail the test instead of stalling the test run
TIMEOUT = 5


def get_test_files(path):
    return sorted(list(path.glob("*.txt")))


def get_test_data(num_tests, len_test, seed=1234):
    # Seeded so a failure can be reproduced
    rng = random.Random(seed)
    separators = [" ", " ", " ", "  ", "\t", "\n", " \t "]

    tests = []
    for i in range(num_tests):
        sequence_size = rng.randint(1, len_test)
        hex_sequence = [f"{rng.randrange(256):02x}" for _ in range(sequence_size)]

        # Remove all "0a"
        hex_sequence = [x for x in hex_sequence if x != "0a"]
        hex_sequence = [x.upper() if rng.random() < 0.25 else x for x in hex_sequence]

        data = "".join(x + rng.choice(separators) for x in hex_sequence).encode()
        tests.append(pytest.param(data, id=str(i)))
    return tests


def run_hex2raw(data, *args, executable=NEW_HEX2RAW):
    return run(
        [executable, *args],
        input=data,
        stdout=PIPE,
        stderr=PIPE,
        timeout=TIMEOUT,
    )


def run_commands(data):
    new_result = run_hex2raw(data, executable=NEW_HEX2RAW)
    old_result = run_hex2raw(data, executable=OLD_HEX2RAW)
    return new_result, old_result


@pytest.mark.parametrize("test_file", get_test_files(Path("tests/same")))
def test_same_output(test_file):
    new_result, old_result = run_commands(test_file.read_bytes())

    # Check if both executables produced the same output
    assert new_result.returncode == old_result.returncode
    assert new_result.stdout == old_result.stdout
    assert new_result.stderr == old_result.stderr


@pytest.mark.parametrize(
    "data",
    [
        b"48 c7\r\nc3\r\n",
        b" ".join([b"41"] * 50_000) + b"\n",
    ],
    ids=["crlf", "line-longer-than-64k"],
)
def test_same_output_inline(data):
    new_result, old_result = run_commands(data)

    assert new_result.returncode == old_result.returncode == 0
    assert new_result.stdout == old_result.stdout
    assert new_result.stderr == old_result.stderr


@pytest.mark.parametrize("test_file", get_test_files(Path("tests/different")))
def test_different(test_file):
    new_result, old_result = run_commands(test_file.read_bytes())

    assert new_result.returncode == 0
    assert old_result.returncode == 255
    assert new_result.stdout == b"this is a test\n"
    assert new_result.stderr == b""


def test_new_line():
    test_file = Path("tests/test-newline.txt")
    new_result, old_result = run_commands(test_file.read_bytes())

    # They both work and return the same thing...
    assert new_result.returncode == old_result.returncode
    assert new_result.stdout == old_result.stdout

    # ...but the new hex2raw returns something to stderr while the other one doesn't
    assert len(new_result.stderr) > 0
    assert len(old_result.stderr) == 0


def test_new_line_warning_is_printed_once():
    result = run_hex2raw(b"0a\n48\n0a 0a\n")

    assert result.returncode == 0
    assert result.stdout == b"\n\x48\n\n\n"
    assert result.stderr.count(b"Warning") == 1
    assert b"lines 1, 3" in result.stderr


@pytest.mark.parametrize("test_data", get_test_data(100, 100))
def test_random(test_data):
    new_result, old_result = run_commands(test_data)

    # Check if both executables produced the same output
    assert new_result.returncode == old_result.returncode
    assert new_result.stdout == old_result.stdout
    assert new_result.stderr == old_result.stderr


@pytest.mark.parametrize(
    "data",
    [
        b"/* forgot to close\n48 c7 c3\n",
        b"48 c7 /* mistyped end *\nc3\n",
        b"/* nested /* comment */\n48 c7 c3\n",
    ],
)
def test_unclosed_comment(data):
    result = run_hex2raw(data)

    assert result.returncode == 1
    assert result.stdout == b""
    assert b"never closed" in result.stderr


@pytest.mark.parametrize(
    "data",
    [
        b"48 c7 // see /* note\nc3\n",
        b"48 c7 // see /* note\n/* x */ c3\n",
        b"48 c7 // see */ note\nc3\n",
        b"48 /* see // note */ c7\nc3\n",
    ],
)
def test_comment_markers_inside_comments(data):
    result = run_hex2raw(data)

    assert result.returncode == 0
    assert result.stdout == b"\x48\xc7\xc3\n"
    assert result.stderr == b""


@pytest.mark.parametrize(
    "data, expected",
    [
        (b"0 0\n", b"\x00\n"),
        (b"ab 19 4 00 00 00 00 0\n", b"\xab\x19\x40\x00\x00\x00\x00\n"),
    ],
)
def test_odd_length_group_warns(data, expected):
    result = run_hex2raw(data)

    assert result.returncode == 0
    assert result.stdout == expected
    assert result.stderr.startswith(b"Warning: line 1:")
    assert result.stderr.count(b"Warning") == 1


def test_odd_number_of_digits_on_line():
    result = run_hex2raw(b"48 c7\n48 c 7c\n")

    assert result.returncode == 1
    assert result.stdout == b""
    assert result.stderr.startswith(b"Error: line 2:")
    assert b'"c"' in result.stderr


@pytest.mark.parametrize(
    "data, message",
    [
        (b"48 c7\n00 00\nzz\n", b'Error: line 3: invalid hex value "zz"'),
        (b"0x48\n", b"0x prefix"),
        (b"48\nc7 */\n", b"Error: line 2: found */ without a matching /*"),
    ],
)
def test_errors(data, message):
    result = run_hex2raw(data)

    assert result.returncode == 1
    # Nothing is written if there is an error, even for the lines that were fine
    assert result.stdout == b""
    assert message in result.stderr


def test_too_many_errors():
    result = run_hex2raw(b"zz\n" * 100)

    assert result.returncode == 1
    assert result.stderr.count(b"Error") == 10
    assert b"Too many errors" in result.stderr


def test_byte_order_mark():
    result = run_hex2raw(b"\xef\xbb\xbf48 c7\n")

    assert result.returncode == 0
    assert result.stdout == b"\x48\xc7\n"
    assert result.stderr == b""


@pytest.fixture
def exploit(tmp_path):
    path = tmp_path / "exploit.txt"
    path.write_bytes(b"48 c7 c3\n")
    return path


def test_input_file(exploit, tmp_path):
    output = tmp_path / "raw.bin"
    result = run_hex2raw(b"", "-i", str(exploit), "-o", str(output))

    assert result.returncode == 0
    assert result.stdout == b""
    assert output.read_bytes() == b"\x48\xc7\xc3\n"


def test_input_file_as_argument(exploit):
    result = run_hex2raw(b"", str(exploit))

    assert result.returncode == 0
    assert result.stdout == b"\x48\xc7\xc3\n"


def test_flags_after_input_file(exploit, tmp_path):
    output = tmp_path / "raw.bin"
    result = run_hex2raw(b"", str(exploit), "-o", str(output))

    assert result.returncode == 0
    assert output.read_bytes() == b"\x48\xc7\xc3\n"


@pytest.mark.parametrize("args", [["{0}", "{0}"], ["-i", "{0}", "{0}"]])
def test_too_many_input_files(exploit, args):
    result = run_hex2raw(b"", *[arg.format(exploit) for arg in args])

    assert result.returncode == 2
    assert result.stdout == b""
    assert result.stderr.startswith(b"Error:")


def test_output_is_input(exploit):
    result = run_hex2raw(b"", "-i", str(exploit), "-o", str(exploit))
    assert result.returncode == 1
    assert exploit.read_bytes() == b"48 c7 c3\n"

    # hex2raw -o exploit.txt < exploit.txt
    with exploit.open("rb") as stdin:
        result = run([NEW_HEX2RAW, "-o", str(exploit)], stdin=stdin, stdout=PIPE, stderr=PIPE, timeout=TIMEOUT)
    assert result.returncode == 1
    assert exploit.read_bytes() == b"48 c7 c3\n"


def test_output_file_untouched_on_error(tmp_path):
    output = tmp_path / "raw.bin"
    output.write_bytes(b"previous output")
    result = run_hex2raw(b"48 c7\nzz\n", "-o", str(output))

    assert result.returncode == 1
    assert output.read_bytes() == b"previous output"


def test_help():
    result = run_hex2raw(b"", "-h")

    assert result.returncode == 0
    assert result.stdout == b""
    assert b"Usage" in result.stderr
