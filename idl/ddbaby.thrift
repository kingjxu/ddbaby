namespace go ddbaby

struct BaseResp {
    1: string StatusMessage;
    2: i32 StatusCode;
}
struct HelloReq {
    1: optional string Name (api.query="name");
}

struct HelloResp {
    1: optional string RespBody;
}

struct DreamExplainReq {
    1: optional string Dream (api.query="dream");
}
struct DreamExplainResp {
    1: optional string ReqID;
    2: optional string Explain;

    255: BaseResp BaseResp;
}

service DDBabyService {
    HelloResp HelloMethod(1: HelloReq req) (api.get="/hello");
    DreamExplainResp DreamExplain(1: DreamExplainReq req) (api.get="/lyxz/dream_explain");
}