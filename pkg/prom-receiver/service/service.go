package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"golang.org/x/sync/errgroup"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appsrv"
	appcommon "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/monitor/notifydrivers/feishu"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/pkg/errors"
)

var (
	Options SOptions
	fsCli   *FeishuNotifier
)

type SOptions struct {
	options.BaseOptions

	AppId     string
	AppSecret string
	Hook      string
}

func StartService() {
	opts := &Options
	options.ParseOptions(opts, os.Args, "prom-receiver.conf", "prom-receiver")

	app := appcommon.InitApp(&opts.BaseOptions, true)
	initHandler(app)
	/*
	 * fsCli, err = NewFeishuNotifier(opts.AppId, opts.AppSecret)
	 * if err != nil {
	 *     log.Fatalf("new feishu client error: %v", err)
	 * }
	 */
	appcommon.ServeForeverWithCleanup(app, &opts.BaseOptions, func() {})
}

func initHandler(app *appsrv.Application) {
	app.AddHandler("POST", "/prometheus/alert", processPromAlert)
}

func processPromAlert(ctx context.Context, rw http.ResponseWriter, r *http.Request) {
	_, _, body := appsrv.FetchEnv(ctx, rw, r)

	if err := FsSend(ctx, Options.Hook, body); err != nil {
		log.Errorf("FsSend error: %v", err)
	}

	appsrv.SendJSON(rw, jsonutils.JSONTrue)
}

type FeishuNotifier struct {
	Client  *feishu.Tenant
	ChatIds []string
}

func NewFeishuNotifier(appId, appSecret string) (*FeishuNotifier, error) {
	cli, err := feishu.NewTenant(appId, appSecret)
	if err != nil {
		return nil, errors.Wrap(err, "new feishu client")
	}
	ret, err := cli.ChatList(0, "")
	if err != nil {
		return nil, err
	}
	chatIds := make([]string, 0)
	for _, obj := range ret.Data.Groups {
		chatIds = append(chatIds, obj.ChatId)
	}
	return &FeishuNotifier{
		Client:  cli,
		ChatIds: chatIds,
	}, nil
}

func FsSend(ctx context.Context, hook string, body jsonutils.JSONObject) error {
	// url := feishu.ApiWebhookRobotSendMessage + hook
	url := "https://open.feishu.cn/open-apis/bot/v2/hook/" + hook

	req := new(PrometheusRequest)
	if err := body.Unmarshal(req); err != nil {
		return errors.Wrapf(err, "unmarshal to prometheus request: %v", err)
	}

	errGrp := errgroup.Group{}
	for idx := range req.Alerts {
		alert := req.Alerts[idx]
		if from := alert.Labels["from"]; from != "loki" {
			log.Infof("--ignore alert %s not from loki", alert.Labels["alertname"])
			continue
		}
		if alert.Status != "firing" {
			log.Infof("--ignore alert %s status %q", alert.Labels["alertname"], alert.Status)
			log.Warningf("ignored alert %s", jsonutils.Marshal(alert))
			continue
		}
		errGrp.Go(func() error {
			log.Debugf("==starting send alert %s", jsonutils.Marshal(alert))
			card, err := genCardByAlert(alert)
			if err != nil {
				return errors.Wrapf(err, "genCardByAlert")
			}
			obj, err := feishu.Request(httputils.POST, url, http.Header{}, jsonutils.Marshal(card))
			if err != nil {
				return err
			}
			resp := new(feishu.WebhookRobotMsgResp)
			err = obj.Unmarshal(resp)
			if !resp.Ok {
				return errors.Errorf("response error, msg: %s", resp.Error)
			}
			return err
		})
	}
	return errGrp.Wait()
}

func (fs *FeishuNotifier) Send(ctx context.Context, body jsonutils.JSONObject) error {
	log.Infof("Sending alert notification to feishu")
	errGrp := errgroup.Group{}
	for _, cId := range fs.ChatIds {
		errGrp.Go(func() error {
			msg, err := genCard(cId, body.PrettyString())
			if err != nil {
				return err
			}
			if _, err := fs.Client.SendMessage(*msg); err != nil {
				log.Errorf("--feishu send msg error: %s, error: %v", jsonutils.Marshal(msg), err)
				return err
			}
			log.Errorf("--feishu send msg: %s", jsonutils.Marshal(msg))
			return nil
		})
	}
	return errGrp.Wait()
}

func getCommonInfoMod(alert PrometheusAlert, labels map[string]string) feishu.CardElement {
	starsAt := alert.StartsAt
	utcTime, err := time.Parse(time.RFC3339Nano, starsAt)
	if err == nil {
		starsAt = utcTime.Local().String()
	} else {
		log.Errorf("parse input startsAt time %q", starsAt)
	}
	elem := feishu.CardElement{
		Tag: feishu.TagDiv,
		// Text: feishu.NewCardElementText(config.Title),
		Fields: []*feishu.CardElementField{
			feishu.NewCardElementTextField(false, fmt.Sprintf("**触发时间:** %s", alert.StartsAt)),
		},
	}
	severity := labels["severity"]
	if severity != "" {
		field := feishu.NewCardElementTextField(false, fmt.Sprintf("**级别:** %s", severity))
		elem.Fields = append(elem.Fields, field)
	}
	return elem
}

func getAlertMod(alert PrometheusAlert, labels map[string]string) *feishu.CardElement {
	elem := feishu.CardElement{
		Tag: feishu.TagDiv,
		Fields: []*feishu.CardElementField{
			feishu.NewCardElementTextField(false, fmt.Sprintf("**容器:** %s/%s/%s", labels["namespace"], labels["pod"], labels["container"])),
		},
	}

	if hostname, ok := labels["hostname"]; ok {
		elem.Fields = append(elem.Fields, feishu.NewCardElementTextField(false, fmt.Sprintf("**节点: ** %s", hostname)))
	}

	annotations := alert.Annotations

	elem.Fields = append(elem.Fields, feishu.NewCardElementTextField(false, fmt.Sprintf("**描述: ** %s", annotations["description"])))
	return &elem
}

func genCardByAlert(alert PrometheusAlert) (*feishu.MsgReq, error) {
	labels := alert.Labels

	commonElem := getCommonInfoMod(alert, labels)

	msg := &feishu.MsgReq{
		MsgType: feishu.MsgTypeInteractive,
		Card: &feishu.Card{
			Config: &feishu.CardConfig{
				WideScreenMode: false,
			},
			Header: &feishu.CardHeader{
				Title: &feishu.CardHeaderTitle{
					Tag:     feishu.TagPlainText,
					Content: labels["alertname"],
				},
			},
			Elements: []interface{}{
				commonElem,
			},
		},
	}

	msElem := getAlertMod(alert, labels)
	msg.Card.Elements = append(msg.Card.Elements, msElem)
	return msg, nil
}

func genCard(chatId string, mdContent string) (*feishu.MsgReq, error) {
	// 消息卡片: https://open.feishu.cn/document/ukTMukTMukTM/uYTNwUjL2UDM14iN1ATN
	msg := &feishu.MsgReq{
		ChatId:  chatId,
		MsgType: feishu.MsgTypeInteractive,
		Card: &feishu.Card{
			Config: &feishu.CardConfig{
				WideScreenMode: false,
			},
			Elements: []interface{}{
				feishu.CardElement{
					Tag: feishu.TagDiv,
					Text: &feishu.CardElement{
						Content: mdContent,
						Tag:     feishu.TagLarkMd,
					},
				},
			},
		},
	}
	return msg, nil
}
