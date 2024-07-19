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

struct PickNameReq {
    1: optional string FamilyName (api.query="family_name");
    2: optional string Gender (api.query="gender");
    3: optional string NameLen (api.query="name_len");
    4: optional string Remark (api.query="remark");

}
struct PickNameResp {
    1: optional string ReqID;
    2: optional string Explain;

    255: BaseResp BaseResp;
}

struct NameFortuneReq {
    1: optional string Name (api.query="name");

}
struct NameFortuneResp {
    1: optional string ReqID;
    2: optional string Explain;

    255: BaseResp BaseResp;
}



service DDBabyService {
    HelloResp HelloMethod(1: HelloReq req) (api.get="/hello");
    DreamExplainResp DreamExplain(1: DreamExplainReq req) (api.get="/lyxz/dream_explain");
    PickNameResp PickName(1: PickNameReq req) (api.get="/lyxz/pick_name");
    NameFortuneResp NameFortune(1: NameFortuneReq req) (api.get="/lyxz/name_fortune");
}