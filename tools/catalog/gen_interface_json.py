#!/usr/bin/env python3
"""从SDK自身的OpenAPI单文件生成权限目录；不访问业务服务、不猜测缺失契约。"""
import argparse
import hashlib
import json
from pathlib import Path

import yaml

class StrictLoader(yaml.SafeLoader):
    """重复键通常覆盖旧权限，因此生成阶段直接失败。"""
    def construct_mapping(self,node,deep=False):
        result={}
        for key_node,value_node in node.value:
            key=self.construct_object(key_node,deep=deep)
            if key in result:
                raise ValueError(f"duplicate YAML key: {key}")
            result[key]=self.construct_object(value_node,deep=deep)
        return result

METHODS={"get","post","put","delete","patch","head","options"}

def generate(source):
    raw=source.read_bytes()
    document=yaml.load(raw,Loader=StrictLoader)
    if not isinstance(document,dict) or not isinstance(document.get("paths"),dict) or not document["paths"]:
        raise ValueError("paths must be a nonempty mapping")
    operations=[]
    ids=set()
    for path,methods in document["paths"].items():
        if not isinstance(path,str) or not path.startswith("/") or not isinstance(methods,dict):
            raise ValueError("malformed path item")
        for method,operation in methods.items():
            if method not in METHODS:
                continue
            if not isinstance(operation,dict) or not isinstance(operation.get("operationId"),str) or not operation["operationId"]:
                raise ValueError("operationId is required")
            operation_id=operation["operationId"]
            if operation_id in ids:
                raise ValueError("duplicate operationId")
            ids.add(operation_id)
            security=operation.get("security",document.get("security",[{"default":[]}]))
            if not isinstance(security,list):
                raise ValueError("security must be an array")
            full=path if path.startswith("/api/v4/") else "/api/v4"+path
            operations.append({"method":method.upper(),"path":full,"operation_id":operation_id,"module":path.strip("/").split("/")[0],"auth_required":bool(security)})
    if not operations:
        raise ValueError("empty operation catalog")
    return {"version":"1","generated_from":"gateapi-go/v7@v7.2.164/openapi.yaml","source_sha256":hashlib.sha256(raw).hexdigest(),"operations":sorted(operations,key=lambda item:(item["path"],item["method"]))}

def main():
    parser=argparse.ArgumentParser()
    parser.add_argument("--src",required=True,type=Path)
    parser.add_argument("--out",required=True,type=Path)
    args=parser.parse_args()
    # 先完整解析、验证，再生成输出；非法输入不留部分目录。
    result=generate(args.src)
    args.out.parent.mkdir(parents=True,exist_ok=True)
    args.out.write_text(json.dumps(result,ensure_ascii=False,indent=2)+"\n")

if __name__=="__main__":
    main()
