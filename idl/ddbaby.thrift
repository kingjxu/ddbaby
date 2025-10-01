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
    1: optional i64 QuestionID;
    2: optional string Content;
    3: optional list<string> Options;
}
struct HealthEvaluateQuestionsReq {
    1: optional string QuestionType (api.query="question_type");
}
struct HealthEvaluateQuestionsResp {
    1: optional list<HealthEvaluateQuestionItem> Questions;

    255: BaseResp BaseResp;
}

struct HealthCreateOrderReq{
    1: optional string ProductID (api.query="product_id");
}
struct HealthCreateOrderResp {
    1: optional string OrderID;
    2: optional string PrepayID;
    255: BaseResp BaseResp;
}
struct HealthDeliveryReq{
    1: optional string ProductID (api.query="product_id");
}
struct HealthDeliveryResp {
    1: optional string OrderID;
    2: optional string PrepayID;
    255: BaseResp BaseResp;
}

struct JkQoItem {
    1: optional i64 id;
    2: optional string question;
    3: optional list<string> options;
}
struct GetJkQoListReq {
    1: optional string qo_type (api.query="qo_type");
    2: optional bool need_pic (api.query="need_pic");
}

struct GetJkQoListResp {
    1: optional string title;
    2: optional string pic;
    3: optional i32 qo_cnt;
    4: optional i32 expect_complete_time;
    5: optional list<JkQoItem> qo;
    6: optional string tips;
    7: optional i32 participant_count;

    255: BaseResp BaseResp;
}

struct QAItem {
    1: optional string question;
    2: optional string answer;
}
struct JkCreateOrderReq {
    1: optional string user_id (api.query="user_id");
    2: optional string qo_type (api.query="qo_type");
    3: optional list<QAItem> qa_items (api.query="qa_items");
    4: optional i32 seq (api.query="seq");
}

struct JkCreateOrderResp {
    1: optional string order_id;
    2: optional string pay_url;

    255: BaseResp BaseResp;
}

struct GetOrderInfoReq  {
    1: optional string order_id (api.query="order_id");
}

struct GetOrderInfoResp {
    1: optional string order_id;
    2: optional string user_id;
    3: optional string product_name;
    4: optional i32 amount;
    5: optional i32 status;
    6: optional i32 seq;
    7: optional i64 create_time;
    8: optional i32 next_amount;

    50: optional string jump_url;
    255: BaseResp BaseResp;
}


struct PayCallbackReq  {
    1: optional string order_id (api.query="user_id");
}

struct PayCallbackResp {
    1: optional string code;
    2: optional string message;

}

service DDBabyService {
    HelloResp HelloMethod(1: HelloReq req) (api.get="/hello");
    DreamExplainResp DreamExplain(1: DreamExplainReq req) (api.get="/lyxz/dream_explain");
    PickNameResp PickName(1: PickNameReq req) (api.get="/lyxz/pick_name");
    NameFortuneResp NameFortune(1: NameFortuneReq req) (api.get="/lyxz/name_fortune");
    TaLuoPredictResp TaLuoPredict(1:TaLuoPredictReq req) (api.get="/lyxz/taluo_predict");

    HealthEvaluateQuestionsResp HealthEvaluateQuestions(1:HealthEvaluateQuestionsReq req) (api.get="/health/questions")
    HealthCreateOrderResp HealthCreateOrder(1:HealthCreateOrderReq req) (api.post="/health/create_order")

    GetJkQoListResp GetJkQoList(1:GetJkQoListReq req) (api.get="/jk/qo_list")
    JkCreateOrderResp JkCreateOrder(1:JkCreateOrderReq req) (api.post="/jk/create_order")
    GetOrderInfoResp GetOrderInfo(1:GetOrderInfoReq req) (api.get="/jk/order_info")
    PayCallbackResp PayCallback(1:PayCallbackReq req) (api.post="/jk/pay_callback")
}