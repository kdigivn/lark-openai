package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"lark-openai/initialization"
	"lark-openai/services"
	"lark-openai/services/openai"
	"lark-openai/utils"

	"github.com/google/uuid"
	larkcard "github.com/larksuite/oapi-sdk-go/v3/card"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

type CardKind string
type CardChatType string

var (
	ClearCardKind      = CardKind("clear")            // 清空上下文
	PicModeChangeKind  = CardKind("pic_mode_change")  // 切换图片创作模式
	PicResolutionKind  = CardKind("pic_resolution")   // 图片分辨率调整
	PicTextMoreKind    = CardKind("pic_text_more")    // 重新根据文本生成图片
	PicVarMoreKind     = CardKind("pic_var_more")     // 变量图片
	RoleTagsChooseKind = CardKind("role_tags_choose") // 内置角色所属标签选择
	RoleChooseKind     = CardKind("role_choose")      // 内置角色选择
	AIModeChooseKind   = CardKind("ai_mode_choose")   // AI模式选择
)

var (
	GroupChatType = CardChatType("group")
	UserChatType  = CardChatType("personal")
)

type CardMsg struct {
	Kind      CardKind
	ChatType  CardChatType
	Value     interface{}
	SessionId string
	MsgId     string
}

type MenuOption struct {
	value string
	label string
}

func replyCard(ctx context.Context,
	msgId *string,
	cardContent string,
) error {
	client := initialization.GetLarkClient()
	resp, err := client.Im.Message.Reply(ctx, larkim.NewReplyMessageReqBuilder().
		MessageId(*msgId).
		Body(larkim.NewReplyMessageReqBodyBuilder().
			MsgType(larkim.MsgTypeInteractive).
			Uuid(uuid.New().String()).
			Content(cardContent).
			Build()).
		Build())

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Println(resp.Code, resp.Msg, resp.RequestId())
		return errors.New(resp.Msg)
	}
	return nil
}

func newSendCard(
	header *larkcard.MessageCardHeader,
	elements ...larkcard.MessageCardElement) (string,
	error) {
	config := larkcard.NewMessageCardConfig().
		WideScreenMode(false).
		EnableForward(true).
		UpdateMulti(false).
		Build()
	var aElementPool []larkcard.MessageCardElement
	for _, element := range elements {
		aElementPool = append(aElementPool, element)
	}
	// 卡片消息体
	cardContent, err := larkcard.NewMessageCard().
		Config(config).
		Header(header).
		Elements(
			aElementPool,
		).
		String()
	return cardContent, err
}

func newSimpleSendCard(
	elements ...larkcard.MessageCardElement) (string,
	error) {
	config := larkcard.NewMessageCardConfig().
		WideScreenMode(false).
		EnableForward(true).
		UpdateMulti(false).
		Build()
	var aElementPool []larkcard.MessageCardElement
	for _, element := range elements {
		aElementPool = append(aElementPool, element)
	}
	// 卡片消息体
	cardContent, err := larkcard.NewMessageCard().
		Config(config).
		Elements(
			aElementPool,
		).
		String()
	return cardContent, err
}

// withSplitLine 用于生成分割线
func withSplitLine() larkcard.MessageCardElement {
	splitLine := larkcard.NewMessageCardHr().
		Build()
	return splitLine
}

// withHeader 用于生成消息头
func withHeader(title string, color string) *larkcard.
	MessageCardHeader {
	if title == "" {
		title = utils.I18n.Sprintf("🤖️nhắc nhở của robot")
	}
	header := larkcard.NewMessageCardHeader().
		Template(color).
		Title(larkcard.NewMessageCardPlainText().
			Content(title).
			Build()).
		Build()
	return header
}

// withNote 用于生成纯文本脚注
func withNote(note string) larkcard.MessageCardElement {
	noteElement := larkcard.NewMessageCardNote().
		Elements([]larkcard.MessageCardNoteElement{larkcard.NewMessageCardPlainText().
			Content(note).
			Build()}).
		Build()
	return noteElement
}

// withMainMd 用于生成markdown消息体
func withMainMd(msg string) larkcard.MessageCardElement {
	msg, i := processMessage(msg)
	msg = processNewLine(msg)
	if i != nil {
		return nil
	}
	mainElement := larkcard.NewMessageCardDiv().
		Fields([]*larkcard.MessageCardField{larkcard.NewMessageCardField().
			Text(larkcard.NewMessageCardLarkMd().
				Content(msg).
				Build()).
			IsShort(true).
			Build()}).
		Build()
	return mainElement
}

// withMainText 用于生成纯文本消息体
func withMainText(msg string) larkcard.MessageCardElement {
	msg, i := processMessage(msg)
	msg = cleanTextBlock(msg)
	if i != nil {
		return nil
	}
	mainElement := larkcard.NewMessageCardDiv().
		Fields([]*larkcard.MessageCardField{larkcard.NewMessageCardField().
			Text(larkcard.NewMessageCardPlainText().
				Content(msg).
				Build()).
			IsShort(false).
			Build()}).
		Build()
	return mainElement
}

func withImageDiv(imageKey string) larkcard.MessageCardElement {
	imageElement := larkcard.NewMessageCardImage().
		ImgKey(imageKey).
		Alt(larkcard.NewMessageCardPlainText().Content("").
			Build()).
		Preview(true).
		Mode(larkcard.MessageCardImageModelCropCenter).
		CompactWidth(true).
		Build()
	return imageElement
}

// withMdAndExtraBtn 用于生成带有额外按钮的消息体
func withMdAndExtraBtn(msg string, btn *larkcard.
	MessageCardEmbedButton) larkcard.MessageCardElement {
	msg, i := processMessage(msg)
	msg = processNewLine(msg)
	if i != nil {
		return nil
	}
	mainElement := larkcard.NewMessageCardDiv().
		Fields(
			[]*larkcard.MessageCardField{
				larkcard.NewMessageCardField().
					Text(larkcard.NewMessageCardLarkMd().
						Content(msg).
						Build()).
					IsShort(true).
					Build()}).
		Extra(btn).
		Build()
	return mainElement
}

func newBtn(content string, value map[string]interface{},
	typename larkcard.MessageCardButtonType) *larkcard.
	MessageCardEmbedButton {
	btn := larkcard.NewMessageCardEmbedButton().
		Type(typename).
		Value(value).
		Text(larkcard.NewMessageCardPlainText().
			Content(content).
			Build())
	return btn
}

func newMenu(
	placeHolder string,
	value map[string]interface{},
	options ...MenuOption,
) *larkcard.
	MessageCardEmbedSelectMenuStatic {
	var aOptionPool []*larkcard.MessageCardEmbedSelectOption
	for _, option := range options {
		aOption := larkcard.NewMessageCardEmbedSelectOption().
			Value(option.value).
			Text(larkcard.NewMessageCardPlainText().
				Content(option.label).
				Build())
		aOptionPool = append(aOptionPool, aOption)

	}
	btn := larkcard.NewMessageCardEmbedSelectMenuStatic().
		MessageCardEmbedSelectMenuStatic(larkcard.NewMessageCardEmbedSelectMenuBase().
			Options(aOptionPool).
			Placeholder(larkcard.NewMessageCardPlainText().
				Content(placeHolder).
				Build()).
			Value(value).
			Build()).
		Build()
	return btn
}

// 清除卡片按钮
func withClearDoubleCheckBtn(sessionID *string) larkcard.MessageCardElement {
	confirmBtn := newBtn(utils.I18n.Sprintf("Bạn có chắc chắn muốn xóa?"), map[string]interface{}{
		"value":     "1",
		"kind":      ClearCardKind,
		"chatType":  UserChatType,
		"sessionId": *sessionID,
	}, larkcard.MessageCardButtonTypeDanger,
	)
	cancelBtn := newBtn(utils.I18n.Sprintf("Tôi sẽ suy nghĩ lại"), map[string]interface{}{
		"value":     "0",
		"kind":      ClearCardKind,
		"sessionId": *sessionID,
		"chatType":  UserChatType,
	},
		larkcard.MessageCardButtonTypeDefault)

	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{confirmBtn, cancelBtn}).
		Layout(larkcard.MessageCardActionLayoutBisected.Ptr()).
		Build()

	return actions
}

func withPicModeDoubleCheckBtn(sessionID *string) larkcard.
	MessageCardElement {
	confirmBtn := newBtn(utils.I18n.Sprintf("Chuyển đổi chế độ"), map[string]interface{}{
		"value":     "1",
		"kind":      PicModeChangeKind,
		"chatType":  UserChatType,
		"sessionId": *sessionID,
	}, larkcard.MessageCardButtonTypeDanger,
	)
	cancelBtn := newBtn(utils.I18n.Sprintf("Tôi sẽ suy nghĩ lại"), map[string]interface{}{
		"value":     "0",
		"kind":      PicModeChangeKind,
		"sessionId": *sessionID,
		"chatType":  UserChatType,
	},
		larkcard.MessageCardButtonTypeDefault)

	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{confirmBtn, cancelBtn}).
		Layout(larkcard.MessageCardActionLayoutBisected.Ptr()).
		Build()

	return actions
}

func withOneBtn(btn *larkcard.MessageCardEmbedButton) larkcard.
	MessageCardElement {
	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{btn}).
		Layout(larkcard.MessageCardActionLayoutFlow.Ptr()).
		Build()
	return actions
}

//新建对话按钮

func withPicResolutionBtn(sessionID *string) larkcard.
	MessageCardElement {
	cancelMenu := newMenu(utils.I18n.Sprintf("Độ phân giải mặc định"),
		map[string]interface{}{
			"value":     "0",
			"kind":      PicResolutionKind,
			"sessionId": *sessionID,
			"msgId":     *sessionID,
		},
		MenuOption{
			label: "256x256",
			value: string(services.Resolution256),
		},
		MenuOption{
			label: "512x512",
			value: string(services.Resolution512),
		},
		MenuOption{
			label: "1024x1024",
			value: string(services.Resolution1024),
		},
	)

	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{cancelMenu}).
		Layout(larkcard.MessageCardActionLayoutFlow.Ptr()).
		Build()
	return actions
}

func withRoleTagsBtn(sessionID *string, tags ...string) larkcard.
	MessageCardElement {
	var menuOptions []MenuOption

	for _, tag := range tags {
		menuOptions = append(menuOptions, MenuOption{
			label: tag,
			value: tag,
		})
	}
	cancelMenu := newMenu(utils.I18n.Sprintf("Chọn phân loại nhân vật"),
		map[string]interface{}{
			"value":     "0",
			"kind":      RoleTagsChooseKind,
			"sessionId": *sessionID,
			"msgId":     *sessionID,
		},
		menuOptions...,
	)

	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{cancelMenu}).
		Layout(larkcard.MessageCardActionLayoutFlow.Ptr()).
		Build()
	return actions
}

func withRoleBtn(sessionID *string, titles ...string) larkcard.
	MessageCardElement {
	var menuOptions []MenuOption

	for _, tag := range titles {
		menuOptions = append(menuOptions, MenuOption{
			label: tag,
			value: tag,
		})
	}
	cancelMenu := newMenu(utils.I18n.Sprintf("Xem các nhân vật tích hợp sẵn"),
		map[string]interface{}{
			"value":     "0",
			"kind":      RoleChooseKind,
			"sessionId": *sessionID,
			"msgId":     *sessionID,
		},
		menuOptions...,
	)

	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{cancelMenu}).
		Layout(larkcard.MessageCardActionLayoutFlow.Ptr()).
		Build()
	return actions
}

func withAIModeBtn(sessionID *string, aiModeStrs []string) larkcard.MessageCardElement {
	var menuOptions []MenuOption
	for _, label := range aiModeStrs {
		menuOptions = append(menuOptions, MenuOption{
			label: label,
			value: label,
		})
	}

	cancelMenu := newMenu(utils.I18n.Sprintf("Chọn chế độ"),
		map[string]interface{}{
			"value":     "0",
			"kind":      AIModeChooseKind,
			"sessionId": *sessionID,
			"msgId":     *sessionID,
		},
		menuOptions...,
	)

	actions := larkcard.NewMessageCardAction().
		Actions([]larkcard.MessageCardActionElement{cancelMenu}).
		Layout(larkcard.MessageCardActionLayoutFlow.Ptr()).
		Build()
	return actions
}

func replyMsg(ctx context.Context, msg string, msgId *string) error {
	msg, i := processMessage(msg)
	if i != nil {
		return i
	}
	client := initialization.GetLarkClient()
	content := larkim.NewTextMsgBuilder().
		Text(msg).
		Build()

	resp, err := client.Im.Message.Reply(ctx, larkim.NewReplyMessageReqBuilder().
		MessageId(*msgId).
		Body(larkim.NewReplyMessageReqBodyBuilder().
			MsgType(larkim.MsgTypeText).
			Uuid(uuid.New().String()).
			Content(content).
			Build()).
		Build())

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Println(resp.Code, resp.Msg, resp.RequestId())
		return errors.New(resp.Msg)
	}
	return nil
}

func uploadImage(base64Str string) (*string, error) {
	imageBytes, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	client := initialization.GetLarkClient()
	resp, err := client.Im.Image.Create(context.Background(),
		larkim.NewCreateImageReqBuilder().
			Body(larkim.NewCreateImageReqBodyBuilder().
				ImageType(larkim.ImageTypeMessage).
				Image(bytes.NewReader(imageBytes)).
				Build()).
			Build())

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Println(resp.Code, resp.Msg, resp.RequestId())
		return nil, errors.New(resp.Msg)
	}
	return resp.Data.ImageKey, nil
}

func replyImage(ctx context.Context, ImageKey *string,
	msgId *string) error {
	//fmt.Println("sendMsg", ImageKey, msgId)

	msgImage := larkim.MessageImage{ImageKey: *ImageKey}
	content, err := msgImage.String()
	if err != nil {
		fmt.Println(err)
		return err
	}
	client := initialization.GetLarkClient()

	resp, err := client.Im.Message.Reply(ctx, larkim.NewReplyMessageReqBuilder().
		MessageId(*msgId).
		Body(larkim.NewReplyMessageReqBodyBuilder().
			MsgType(larkim.MsgTypeImage).
			Uuid(uuid.New().String()).
			Content(content).
			Build()).
		Build())

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Println(resp.Code, resp.Msg, resp.RequestId())
		return errors.New(resp.Msg)
	}
	return nil
}

func replayImageCardByBase64(ctx context.Context, base64Str string,
	msgId *string, sessionId *string, question string) error {
	imageKey, err := uploadImage(base64Str)
	if err != nil {
		return err
	}
	//example := "img_v2_041b28e3-5680-48c2-9af2-497ace79333g"
	//imageKey := &example
	//fmt.Println("imageKey", *imageKey)
	err = sendImageCard(ctx, *imageKey, msgId, sessionId, question)
	if err != nil {
		return err
	}
	return nil
}

func replayImagePlainByBase64(ctx context.Context, base64Str string,
	msgId *string) error {
	imageKey, err := uploadImage(base64Str)
	if err != nil {
		return err
	}
	//example := "img_v2_041b28e3-5680-48c2-9af2-497ace79333g"
	//imageKey := &example
	//fmt.Println("imageKey", *imageKey)
	err = replyImage(ctx, imageKey, msgId)
	if err != nil {
		return err
	}
	return nil
}

func replayVariantImageByBase64(ctx context.Context, base64Str string,
	msgId *string, sessionId *string) error {
	imageKey, err := uploadImage(base64Str)
	if err != nil {
		return err
	}
	//example := "img_v2_041b28e3-5680-48c2-9af2-497ace79333g"
	//imageKey := &example
	//fmt.Println("imageKey", *imageKey)
	err = sendVarImageCard(ctx, *imageKey, msgId, sessionId)
	if err != nil {
		return err
	}
	return nil
}

func sendMsg(ctx context.Context, msg string, chatId *string) error {
	//fmt.Println("sendMsg", msg, chatId)
	msg, i := processMessage(msg)
	if i != nil {
		return i
	}
	client := initialization.GetLarkClient()
	content := larkim.NewTextMsgBuilder().
		Text(msg).
		Build()

	//fmt.Println("content", content)

	resp, err := client.Im.Message.Create(ctx, larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			MsgType(larkim.MsgTypeText).
			ReceiveId(*chatId).
			Content(content).
			Build()).
		Build())

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Println(resp.Code, resp.Msg, resp.RequestId())
		return errors.New(resp.Msg)
	}
	return nil
}

func sendClearCacheCheckCard(ctx context.Context,
	sessionId *string, msgId *string) {
	newCard, _ := newSendCard(
		withHeader(utils.I18n.Sprintf("🆑 Lời nhắc của robot"), larkcard.TemplateBlue),
		withMainMd(utils.I18n.Sprintf("Bạn có chắc chắn muốn xóa bối cảnh cuộc trò chuyện không?")),
		withNote(utils.I18n.Sprintf("Vui lòng lưu ý rằng điều này sẽ bắt đầu một cuộc trò chuyện mới và bạn sẽ không thể tận dụng thông tin lịch sử từ các chủ đề trước đó")),
		withClearDoubleCheckBtn(sessionId))
	replyCard(ctx, msgId, newCard)
}

func sendSystemInstructionCard(ctx context.Context,
	sessionId *string, msgId *string, content string) {
	newCard, _ := newSendCard(
		withHeader(utils.I18n.Sprintf("🥷  Đã chuyển sang chế độ nhập vai"), larkcard.TemplateIndigo),
		withMainText(content),
		withNote(utils.I18n.Sprintf("Vui lòng lưu ý rằng điều này sẽ bắt đầu một cuộc trò chuyện mới và bạn sẽ không thể tận dụng thông tin lịch sử từ các chủ đề trước đó")))
	replyCard(ctx, msgId, newCard)
}

func sendPicCreateInstructionCard(ctx context.Context,
	sessionId *string, msgId *string) {
	newCard, _ := newSendCard(
		withHeader(utils.I18n.Sprintf("🖼️ Đã chuyển sang chế độ tạo hình ảnh"), larkcard.TemplateBlue),
		withPicResolutionBtn(sessionId),
		withNote(utils.I18n.Sprintf("Nhắc nhở: Trả lời bằng văn bản hoặc hình ảnh để cho trí tuệ nhân tạo tạo ra những hình ảnh liên quan.")))
	replyCard(ctx, msgId, newCard)
}

func sendPicModeCheckCard(ctx context.Context,
	sessionId *string, msgId *string) {
	newCard, _ := newSendCard(
		withHeader(utils.I18n.Sprintf("🖼️ lời nhắc của robot"), larkcard.TemplateBlue),
		withMainMd(utils.I18n.Sprintf("Nhận được hình ảnh, chuyển sang chế độ tạo hình ảnh?")),
		withNote(utils.I18n.Sprintf("Vui lòng lưu ý rằng điều này sẽ bắt đầu một cuộc trò chuyện mới và bạn sẽ không thể tận dụng thông tin lịch sử từ các chủ đề trước đó")),
		withPicModeDoubleCheckBtn(sessionId))
	replyCard(ctx, msgId, newCard)
}

func sendNewTopicCard(ctx context.Context,
	sessionId *string, msgId *string, content string) {
	newCard, _ := newSendCard(
		withHeader(utils.I18n.Sprintf("👻️ Đã mở chủ đề mới"), larkcard.TemplateBlue),
		withMainText(content),
		withNote(utils.I18n.Sprintf("Nhắc nhở: Nhấp vào hộp chat để tham gia trả lời và duy trì tính liên tục của chủ đề")))
	replyCard(ctx, msgId, newCard)
}

func sendHelpCard(ctx context.Context,
	sessionId *string, msgId *string) {
	newCard, _ := newSendCard(
		withHeader(utils.I18n.Sprintf("🎒Bạn cần giúp đỡ không?"), larkcard.TemplateBlue),
		withMainMd(utils.I18n.Sprintf("**Tôi là KDIGI Bot，một trò chuyện trí tuệ nhân tạo thông minh dựa trên công nghệ ChatGPT!**")),
		withSplitLine(),
		withMdAndExtraBtn(
			utils.I18n.Sprintf("** 🆑 Xóa bối cảnh cuộc trò chuyện**\nTrả lời bằng văn bản *Xóa* hoặc */clear*"),
			newBtn(utils.I18n.Sprintf("Xoá tất cả"), map[string]interface{}{
				"value":     "1",
				"kind":      ClearCardKind,
				"chatType":  UserChatType,
				"sessionId": *sessionId,
			}, larkcard.MessageCardButtonTypeDanger)),
		withSplitLine(),
		withMainMd(utils.I18n.Sprintf("🤖 **Lựa chọn chế độ trí tuệ nhân tạo** \n"+" Phản hồi bằng văn bản *Chế độ trí tuệ nhân tạo* hoặc */ai_mode*")),
		withSplitLine(),
		withMainMd(utils.I18n.Sprintf("🛖 **Danh sách nhân vật tích hợp sẵn** \n"+" Trả lời bằng văn bản *Danh sách nhân vật* hoặc */roles*")),
		withSplitLine(),
		withMainMd(utils.I18n.Sprintf("🥷 **Chế độ nhập vai**\nPhản hồi bằng văn bản cho chế độ nhập vai hoặc */system* + dấu cách + thông tin nhân vật")),
		withSplitLine(),
		withMainMd(utils.I18n.Sprintf("🎤 **Trò chuyện giọng nói trí tuệ nhân tạo**\nGửi tin nhắn thoại trực tiếp trong chế độ trò chuyện riêng tư")),
		withSplitLine(),
		withMainMd(utils.I18n.Sprintf("🎨 **Chế độ tạo hình ảnh**\nPhản hồi *Chỉnh sửa hình ảnh* hoặc */picture*")),
		withSplitLine(),
		withMainMd(utils.I18n.Sprintf("🎰 **Tra cứu số dư Token**\nPhản hồi *Số dư* hoặc */balance*")),
		withSplitLine(),
		withMainMd(utils.I18n.Sprintf("🔃️ **Quay lại một chủ đề trò chuyện trước đó** 🚧\n"+" Truy cập trang chi tiết phản hồi của chủ đề, phản hồi bằng văn bản *Khôi phục* hoặc */reload*")),
		withSplitLine(),
		withMainMd(utils.I18n.Sprintf("📤 **Xuất dữ liệu trò chuyện** 🚧\n"+" Trả lời bằng văn bản *Xuất ra* hoặc */export*")),
		withSplitLine(),
		withMainMd(utils.I18n.Sprintf("🎰 **Chế độ trò chuyện liên tục và đa chủ đề**\n"+" Nhấp vào hộp trò chuyện để tham gia phản hồi và giữ cho cuộc trò chuyện liên tục. Đồng thời, đặt câu hỏi riêng có thể bắt đầu một chủ đề hoàn toàn mới")),
		withSplitLine(),
		withMainMd(utils.I18n.Sprintf("🎒 **cần thêm sự trợ giúp**\nPhản hồi bằng văn bản *trợ giúp* hoặc */help*")),
	)
	replyCard(ctx, msgId, newCard)
}

func sendImageCard(ctx context.Context, imageKey string,
	msgId *string, sessionId *string, question string) error {
	newCard, _ := newSimpleSendCard(
		withImageDiv(imageKey),
		withSplitLine(),
		//再来一张
		withOneBtn(newBtn(utils.I18n.Sprintf("thêm một tấm (ảnh/hình)"), map[string]interface{}{
			"value":     question,
			"kind":      PicTextMoreKind,
			"chatType":  UserChatType,
			"msgId":     *msgId,
			"sessionId": *sessionId,
		}, larkcard.MessageCardButtonTypePrimary)),
	)
	replyCard(ctx, msgId, newCard)
	return nil
}

func sendVarImageCard(ctx context.Context, imageKey string,
	msgId *string, sessionId *string) error {
	newCard, _ := newSimpleSendCard(
		withImageDiv(imageKey),
		withSplitLine(),
		//再来一张
		withOneBtn(newBtn(utils.I18n.Sprintf("thêm một tấm (ảnh/hình)"), map[string]interface{}{
			"value":     imageKey,
			"kind":      PicVarMoreKind,
			"chatType":  UserChatType,
			"msgId":     *msgId,
			"sessionId": *sessionId,
		}, larkcard.MessageCardButtonTypePrimary)),
	)
	replyCard(ctx, msgId, newCard)
	return nil
}

func sendBalanceCard(ctx context.Context, msgId *string,
	balance openai.BalanceResponse) {
	newCard, _ := newSendCard(
		withHeader(utils.I18n.Sprintf("🎰️ Tra cứu số dư"), larkcard.TemplateBlue),
		withMainMd(utils.I18n.Sprintf("tổng hạn mức: %.2f$", balance.TotalGranted)),
		withMainMd(utils.I18n.Sprintf("hạn mức đã sử dụng: %.2f$", balance.TotalUsed)),
		withMainMd(utils.I18n.Sprintf("hạn mức khả dụng: %.2f$", balance.TotalAvailable)),
		withNote(utils.I18n.Sprintf("Ngày hết hạn: %s - %s",
			balance.EffectiveAt.Format("2006-01-02 15:04:05"),
			balance.ExpiresAt.Format("2006-01-02 15:04:05"))),
	)
	replyCard(ctx, msgId, newCard)
}

func SendRoleTagsCard(ctx context.Context,
	sessionId *string, msgId *string, roleTags []string) {
	newCard, _ := newSendCard(
		withHeader(utils.I18n.Sprintf("🛖 Vui lòng chọn danh mục vai trò"), larkcard.TemplateIndigo),
		withRoleTagsBtn(sessionId, roleTags...),
		withNote(utils.I18n.Sprintf("Nhắc nhở: Chọn danh mục mà vai trò thuộc về để chúng tôi có thể đề xuất cho bạn thêm nhiều vai trò liên quan hơn.")))
	replyCard(ctx, msgId, newCard)
}

func SendRoleListCard(ctx context.Context,
	sessionId *string, msgId *string, roleTag string, roleList []string) {
	newCard, _ := newSendCard(
		withHeader(utils.I18n.Sprintf("🛖 danh sách vai trò")+" - "+roleTag, larkcard.TemplateIndigo),
		withRoleBtn(sessionId, roleList...),
		withNote(utils.I18n.Sprintf("Nhắc nhở: Chọn một cảnh quan có sẵn để nhanh chóng nhập vai và bắt đầu chế độ đóng vai.")))
	replyCard(ctx, msgId, newCard)
}

func SendAIModeListsCard(ctx context.Context,
	sessionId *string, msgId *string, aiModeStrs []string) {
	newCard, _ := newSendCard(
		withHeader(utils.I18n.Sprintf("🤖 Lựa chọn chế độ trí tuệ nhân tạo"), larkcard.TemplateIndigo),
		withAIModeBtn(sessionId, aiModeStrs),
		withNote(utils.I18n.Sprintf("Nhắc nhở: Chọn một chế độ có sẵn để giúp trí tuệ nhân tạo hiểu rõ hơn nhu cầu của bạn.")))
	replyCard(ctx, msgId, newCard)
}
