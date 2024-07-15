namespace go ddbaby

struct HelloReq {
    1: string Name (api.query="name");
}

struct HelloResp {
    1: string RespBody;
}

struct DreamExplainReq {
    1: string Dream (api.query="dream");
}
struct DreamExplainResp {
    1: string DreamID;
    2: string Explain;
}

service DDBabyService {
    HelloResp HelloMethod(1: HelloReq req) (api.get="/hello");
    DreamExplainResp DreamExplain(1: DreamExplainReq req) (api.get="/dream_explain");
}