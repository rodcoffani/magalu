# noqa: T201

import os
from typing import Dict, Any, List, TypedDict, TextIO, Tuple
import argparse
import yaml
import json
import re

OAPISchema = Dict[str, Any]


class IndexModule(TypedDict):
    name: str
    url: str
    path: str
    version: str
    description: str


IndexModules = List[IndexModule]


class IndexFile(TypedDict):
    version: str
    modules: IndexModules


modname_re = re.compile("^(?P<name>[a-z0-9-]+)[.]openapi[.]yaml$")
index_filename = "index.openapi.yaml"
index_version = "1.0.0"


def load_yaml(path: str) -> OAPISchema:
    with open(path, "r") as fd:
        return yaml.load(fd, Loader=yaml.FullLoader)


def save_index(mods: IndexModules, path: str):
    with open(os.path.join(path, index_filename), "w") as fd:
        idx_file = IndexFile(version=index_version, modules=mods)
        yaml.dump(idx_file, fd, indent=4, allow_unicode=True)
        return idx_file


def load_mods(
    oapiDir: str, outDir: str | None = None
) -> Tuple[Dict[str, OAPISchema], IndexModules]:
    if outDir is None:
        outDir = oapiDir

    full_mods = {}
    mods = []
    for filename in sorted(os.listdir(oapiDir)):
        if filename == index_filename:
            continue
        match = modname_re.match(filename)
        if not match:
            if filename != index_filename:
                print("ignored file:", filename)
            continue

        filepath = os.path.join(oapiDir, filename)
        relpath = os.path.relpath(filepath, outDir)

        data = load_yaml(filepath)
        info = data["info"]
        url = data["$id"]
        name = match.group("name")
        full_mods[filename] = data
        description = info.get("x-mgc-description", info.get("description", ""))
        mods.append(
            IndexModule(
                name=name,
                url=url,
                path=relpath,
                description=description,
                version=info.get("version", ""),
                summary=info.get("summary", description),
            )
        )
    return full_mods, mods


embed_json_opts = {
    "separators": (",", ":"),
    "ensure_ascii": False,
    "sort_keys": True,
}


def save_embed(
    idx_file: IndexFile,
    full_mods: Dict[str, OAPISchema],
    out: TextIO,
) -> None:
    out.write(
        """\
// Code generated by oapi_index_gen. DO NOT EDIT.

//go:build embed

//nolint

package openapi

import (
\t"os"
\t"syscall"
\t"magalu.cloud/core/dataloader"
)

type embedLoader map[string][]byte

func GetEmbedLoader() dataloader.Loader {
\treturn embedLoaderInstance
}

func (f embedLoader) Load(name string) ([]byte, error) {
\tif data, ok := embedLoaderInstance[name]; ok {
\t\treturn data, nil
\t}
\treturn nil, &os.PathError{Op: "open", Path: name, Err: syscall.ENOENT}
}

func (f embedLoader) String() string {
\treturn "embedLoader"
}

var embedLoaderInstance = embedLoader{
"""
    )

    def add_str(v):
        json.dump(v, out, **embed_json_opts)

    def add_embed(k, v):
        out.write("\t")
        add_str(k)
        out.write(": ([]byte)(")
        # TODO: cleanup embedded documents, remove unused stuff (examples?)
        # and consolidate x-mgc-XXX into final fields, if we're using a single
        # x-mgc- for both CLI and TerraForm
        add_str(json.dumps(v, **embed_json_opts))
        out.write("),\n")

    files = [(index_filename, idx_file)]
    files.extend(sorted(full_mods.items()))
    for k, v in files:
        add_embed(k, v)

    out.write(
        """\
}
"""
    )


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Generate index file for all OAPI YAML files in directory",
    )
    parser.add_argument(
        "dir",
        type=str,
        help="Directory of openapi files",
    )
    parser.add_argument(
        "-o",
        "--output",
        type=str,
        help="Directory to save the new index YAML. Defaults to openapi directory",
    )
    parser.add_argument(
        "--embed",
        type=argparse.FileType("w"),
        help="Write Golang embed loader file",
        default=None,
    )
    args = parser.parse_args()

    full_mods, mods = load_mods(args.dir, args.output)
    print("indexed modules:")
    for mod in mods:
        print(mod)

    idx_file = save_index(mods, args.output or args.dir)
    if args.embed:
        save_embed(idx_file, full_mods, args.embed)
