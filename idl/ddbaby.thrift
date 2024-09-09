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
    1: optional string FamilyName (api.query="family_name"); // 姓氏
    2: optional string Gender (api.query="gender"); // 性别
    3: optional i32 NameLen (api.query="name_len"); // 名字长度
    4: optional string Remark (api.query="remark"); // 备注

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

struct TaLuoPredictReq {
    1: optional string Query (api.query="query");

}
struct TaLuoPredictResp {
    1: optional string ReqID;
    2: optional string Explain;

    255: BaseResp BaseResp;
}

struct HealthEvaluateQuestionItem {
    1: optional string QuestionID;
    2: optional string Content;
    3: optional list<string> Options;
}
struct HealthEvaluateQuestionsReq {
    1: optional string CatetoryName (api.query="catetory_name");
}
struct HealthEvaluateQuestionsResp {
    1: optional list<HealthEvaluateQuestionItem> Questions;

    255: BaseResp BaseResp;
}

service DDBabyService {
    HelloResp HelloMethod(1: HelloReq req) (api.get="/hello");
    DreamExplainResp DreamExplain(1: DreamExplainReq req) (api.get="/lyxz/dream_explain");
    PickNameResp PickName(1: PickNameReq req) (api.get="/lyxz/pick_name");
    NameFortuneResp NameFortune(1: NameFortuneReq req) (api.get="/lyxz/name_fortune");
    TaLuoPredictResp TaLuoPredict(1:TaLuoPredictReq req) (api.get="/lyxz/taluo_predict");

    HealthEvaluateQuestionsResp HealthEvaluateQuestions(1:HealthEvaluateQuestionsReq req) (api.get="/health/questions")

}