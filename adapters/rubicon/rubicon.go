package rubicon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/golang/glog"

	"github.com/prebid/prebid-server/pbs"

	"golang.org/x/net/context/ctxhttp"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type RubiconAdapter struct {
	http         *adapters.HTTPAdapter
	URI          string
	XAPIUsername string
	XAPIPassword string
}

// used for cookies and such
func (a *RubiconAdapter) Name() string {
	return "rubicon"
}

func (a *RubiconAdapter) SkipNoCookies() bool {
	return false
}

type rubiconParams struct {
	AccountId int                `json:"accountId"`
	SiteId    int                `json:"siteId"`
	ZoneId    int                `json:"zoneId"`
	Inventory json.RawMessage    `json:"inventory"`
	Visitor   json.RawMessage    `json:"visitor"`
	Video     rubiconVideoParams `json:"video"`
}

type rubiconImpExtRPTrack struct {
	Mint        string `json:"mint"`
	MintVersion string `json:"mint_version"`
}

type rubiconImpExtRP struct {
	ZoneID int                  `json:"zone_id"`
	Target json.RawMessage      `json:"target"`
	Track  rubiconImpExtRPTrack `json:"track"`
}

type rubiconImpExt struct {
	RP rubiconImpExtRP `json:"rp"`
}

type rubiconUserExtRP struct {
	Target json.RawMessage `json:"target"`
}

type rubiconExtUserTpID struct {
	Source string `json:"source"`
	UID    string `json:"uid"`
}

type rubiconUserExt struct {
	Consent   string                        `json:"consent,omitempty"`
	DigiTrust *openrtb_ext.ExtUserDigiTrust `json:"digitrust"`
	Eids      []openrtb_ext.ExtUserEid      `json:"eids,omitempty"`
	TpID      []rubiconExtUserTpID          `json:"tpid,omitempty"`
	RP        rubiconUserExtRP              `json:"rp"`
}

type rubiconSiteExtRP struct {
	SiteID int `json:"site_id"`
}

type rubiconSiteExt struct {
	RP rubiconSiteExtRP `json:"rp"`
}

type rubiconPubExtRP struct {
	AccountID int `json:"account_id"`
}

type rubiconPubExt struct {
	RP rubiconPubExtRP `json:"rp"`
}

type rubiconBannerExtRP struct {
	SizeID     int    `json:"size_id,omitempty"`
	AltSizeIDs []int  `json:"alt_size_ids,omitempty"`
	MIME       string `json:"mime"`
}

type rubiconBannerExt struct {
	RP rubiconBannerExtRP `json:"rp"`
}

// ***** Video Extension *****
type rubiconVideoParams struct {
	Language     string `json:"language,omitempty"`
	PlayerHeight int    `json:"playerHeight,omitempty"`
	PlayerWidth  int    `json:"playerWidth,omitempty"`
	VideoSizeID  int    `json:"size_id,omitempty"`
	Skip         int    `json:"skip,omitempty"`
	SkipDelay    int    `json:"skipdelay,omitempty"`
}

type rubiconVideoExt struct {
	Skip      int               `json:"skip,omitempty"`
	SkipDelay int               `json:"skipdelay,omitempty"`
	RP        rubiconVideoExtRP `json:"rp"`
}

type rubiconVideoExtRP struct {
	SizeID int `json:"size_id,omitempty"`
}

type rubiconTargetingExt struct {
	RP rubiconTargetingExtRP `json:"rp"`
}

type rubiconTargetingExtRP struct {
	Targeting []rubiconTargetingObj `json:"targeting"`
}

type rubiconTargetingObj struct {
	Key    string   `json:"key"`
	Values []string `json:"values"`
}

type rubiconDeviceExtRP struct {
	PixelRatio float64 `json:"pixelratio"`
}

type rubiconDeviceExt struct {
	RP rubiconDeviceExtRP `json:"rp"`
}

type rubiconUser struct {
	Language string `json:"language"`
}

type rubiSize struct {
	w uint16
	h uint16
}

var rubiSizeMap = map[rubiSize]int{
	{w: 468, h: 60}:    1,
	{w: 728, h: 90}:    2,
	{w: 728, h: 91}:    2,
	{w: 120, h: 600}:   8,
	{w: 160, h: 600}:   9,
	{w: 300, h: 600}:   10,
	{w: 300, h: 250}:   15,
	{w: 300, h: 251}:   15,
	{w: 336, h: 280}:   16,
	{w: 300, h: 100}:   19,
	{w: 980, h: 120}:   31,
	{w: 250, h: 360}:   32,
	{w: 180, h: 500}:   33,
	{w: 980, h: 150}:   35,
	{w: 468, h: 400}:   37,
	{w: 930, h: 180}:   38,
	{w: 320, h: 50}:    43,
	{w: 300, h: 50}:    44,
	{w: 300, h: 300}:   48,
	{w: 300, h: 1050}:  54,
	{w: 970, h: 90}:    55,
	{w: 970, h: 250}:   57,
	{w: 1000, h: 90}:   58,
	{w: 320, h: 80}:    59,
	{w: 1000, h: 1000}: 61,
	{w: 640, h: 480}:   65,
	{w: 320, h: 480}:   67,
	{w: 1800, h: 1000}: 68,
	{w: 320, h: 320}:   72,
	{w: 320, h: 160}:   73,
	{w: 980, h: 240}:   78,
	{w: 980, h: 300}:   79,
	{w: 980, h: 400}:   80,
	{w: 480, h: 300}:   83,
	{w: 970, h: 310}:   94,
	{w: 970, h: 210}:   96,
	{w: 480, h: 320}:   101,
	{w: 768, h: 1024}:  102,
	{w: 480, h: 280}:   103,
	{w: 320, h: 240}:   108,
	{w: 1000, h: 300}:  113,
	{w: 320, h: 100}:   117,
	{w: 800, h: 250}:   125,
	{w: 200, h: 600}:   126,
	{w: 640, h: 320}:   156,
}

//MAS algorithm
func findPrimary(alt []int) (int, []int) {
	min, pos, primary := 0, 0, 0
	for i, size := range alt {
		if size == 15 {
			primary = 15
			pos = i
			break
		} else if size == 2 {
			primary = 2
			pos = i
		} else if size == 9 && primary != 2 {
			primary = 9
			pos = i
		} else if size < alt[min] {
			min = i
		}
	}
	if primary == 0 {
		primary = alt[min]
		pos = min
	}

	alt = append(alt[:pos], alt[pos+1:]...)
	return primary, alt
}

func parseRubiconSizes(sizes []openrtb.Format) (primary int, alt []int, err error) {
	// Fixes #317
	if len(sizes) < 1 {
		err = &errortypes.BadInput{
			Message: "rubicon imps must have at least one imp.format element",
		}
		return
	}
	for _, size := range sizes {
		if rs, ok := rubiSizeMap[rubiSize{w: uint16(size.W), h: uint16(size.H)}]; ok {
			alt = append(alt, rs)
		}
	}
	if len(alt) > 0 {
		primary, alt = findPrimary(alt)
	} else {
		err = &errortypes.BadInput{
			Message: "No primary size found",
		}
	}
	return
}

func (a *RubiconAdapter) callOne(ctx context.Context, reqJSON bytes.Buffer) (result adapters.CallOneResult, err error) {
	httpReq, err := http.NewRequest("POST", a.URI, &reqJSON)
	httpReq.Header.Add("Content-Type", "application/json;charset=utf-8")
	httpReq.Header.Add("Accept", "application/json")
	httpReq.Header.Add("User-Agent", "prebid-server/1.0")
	httpReq.SetBasicAuth(a.XAPIUsername, a.XAPIPassword)

	rubiResp, e := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if e != nil {
		err = e
		return
	}

	defer rubiResp.Body.Close()
	body, _ := ioutil.ReadAll(rubiResp.Body)
	result.ResponseBody = string(body)

	result.StatusCode = rubiResp.StatusCode

	if rubiResp.StatusCode == 204 {
		return
	}

	if rubiResp.StatusCode == http.StatusBadRequest {
		err = &errortypes.BadInput{
			Message: fmt.Sprintf("HTTP status %d; body: %s", rubiResp.StatusCode, result.ResponseBody),
		}
	}

	if rubiResp.StatusCode != http.StatusOK {
		err = &errortypes.BadServerResponse{
			Message: fmt.Sprintf("HTTP status %d; body: %s", rubiResp.StatusCode, result.ResponseBody),
		}
		return
	}

	var bidResp openrtb.BidResponse
	err = json.Unmarshal(body, &bidResp)
	if err != nil {
		err = &errortypes.BadServerResponse{
			Message: err.Error(),
		}
		return
	}
	if len(bidResp.SeatBid) == 0 {
		return
	}
	if len(bidResp.SeatBid[0].Bid) == 0 {
		return
	}
	bid := bidResp.SeatBid[0].Bid[0]

	result.Bid = &pbs.PBSBid{
		AdUnitCode:  bid.ImpID,
		Price:       bid.Price,
		Adm:         bid.AdM,
		Creative_id: bid.CrID,
		// for video, the width and height are undefined as there's no corresponding return value from XAPI
		Width:  bid.W,
		Height: bid.H,
		DealId: bid.DealID,
	}

	// Pull out any server-side determined targeting
	var rpExtTrg rubiconTargetingExt

	if err := json.Unmarshal([]byte(bid.Ext), &rpExtTrg); err == nil {
		// Converting string => array(string) to string => string
		targeting := make(map[string]string)

		// Only pick off the first for now
		for _, target := range rpExtTrg.RP.Targeting {
			targeting[target.Key] = target.Values[0]
		}

		result.Bid.AdServerTargeting = targeting
	}

	return
}

type callOneObject struct {
	requestJson bytes.Buffer
	mediaType   pbs.MediaType
}

func (a *RubiconAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	callOneObjects := make([]callOneObject, 0, len(bidder.AdUnits))
	supportedMediaTypes := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER, pbs.MEDIA_TYPE_VIDEO}

	rubiReq, err := adapters.MakeOpenRTBGeneric(req, bidder, a.Name(), supportedMediaTypes)
	if err != nil {
		return nil, err
	}

	rubiReqImpCopy := rubiReq.Imp

	for i, unit := range bidder.AdUnits {
		// Fixes some segfaults. Since this is legacy code, I'm not looking into it too deeply
		if len(rubiReqImpCopy) <= i {
			break
		}
		// Only grab this ad unit
		// Not supporting multi-media-type add-unit yet
		thisImp := rubiReqImpCopy[i]

		// Amend it with RP-specific information
		var params rubiconParams
		err = json.Unmarshal(unit.Params, &params)
		if err != nil {
			return nil, &errortypes.BadInput{
				Message: err.Error(),
			}
		}

		var mint, mintVersion string
		mint = "prebid"
		mintVersion = req.SDK.Source + "_" + req.SDK.Platform + "_" + req.SDK.Version
		track := rubiconImpExtRPTrack{Mint: mint, MintVersion: mintVersion}

		impExt := rubiconImpExt{RP: rubiconImpExtRP{
			ZoneID: params.ZoneId,
			Target: params.Inventory,
			Track:  track,
		}}
		thisImp.Ext, err = json.Marshal(&impExt)
		if err != nil {
			continue
		}

		// Copy the $.user object and amend with $.user.ext.rp.target
		// Copy avoids race condition since it points to ref & shared with other adapters
		userCopy := *rubiReq.User
		userExt := rubiconUserExt{RP: rubiconUserExtRP{Target: params.Visitor}}
		userCopy.Ext, err = json.Marshal(&userExt)
		// Assign back our copy
		rubiReq.User = &userCopy

		deviceCopy := *rubiReq.Device
		deviceExt := rubiconDeviceExt{RP: rubiconDeviceExtRP{PixelRatio: rubiReq.Device.PxRatio}}
		deviceCopy.Ext, err = json.Marshal(&deviceExt)
		rubiReq.Device = &deviceCopy

		if thisImp.Video != nil {
			videoExt := rubiconVideoExt{Skip: params.Video.Skip, SkipDelay: params.Video.SkipDelay, RP: rubiconVideoExtRP{SizeID: params.Video.VideoSizeID}}
			thisImp.Video.Ext, err = json.Marshal(&videoExt)
		} else {
			primarySizeID, altSizeIDs, err := parseRubiconSizes(unit.Sizes)
			if err != nil {
				continue
			}
			bannerExt := rubiconBannerExt{RP: rubiconBannerExtRP{SizeID: primarySizeID, AltSizeIDs: altSizeIDs, MIME: "text/html"}}
			thisImp.Banner.Ext, err = json.Marshal(&bannerExt)
		}

		siteExt := rubiconSiteExt{RP: rubiconSiteExtRP{SiteID: params.SiteId}}
		pubExt := rubiconPubExt{RP: rubiconPubExtRP{AccountID: params.AccountId}}
		var rubiconUser rubiconUser
		err = json.Unmarshal(req.PBSUser, &rubiconUser)

		if rubiReq.Site != nil {
			siteCopy := *rubiReq.Site
			siteCopy.Ext, err = json.Marshal(&siteExt)
			siteCopy.Publisher = &openrtb.Publisher{}
			siteCopy.Publisher.Ext, err = json.Marshal(&pubExt)
			siteCopy.Content = &openrtb.Content{}
			siteCopy.Content.Language = rubiconUser.Language
			rubiReq.Site = &siteCopy
		} else {
			site := &openrtb.Site{}
			site.Content = &openrtb.Content{}
			site.Content.Language = rubiconUser.Language
			rubiReq.Site = site
		}

		if rubiReq.App != nil {
			appCopy := *rubiReq.App
			appCopy.Ext, err = json.Marshal(&siteExt)
			appCopy.Publisher = &openrtb.Publisher{}
			appCopy.Publisher.Ext, err = json.Marshal(&pubExt)
			rubiReq.App = &appCopy
		}

		rubiReq.Imp = []openrtb.Imp{thisImp}

		var reqBuffer bytes.Buffer
		err = json.NewEncoder(&reqBuffer).Encode(rubiReq)
		if err != nil {
			return nil, err
		}
		callOneObjects = append(callOneObjects, callOneObject{reqBuffer, unit.MediaTypes[0]})
	}
	if len(callOneObjects) == 0 {
		return nil, &errortypes.BadInput{
			Message: "Invalid ad unit/imp",
		}
	}

	ch := make(chan adapters.CallOneResult)
	for _, obj := range callOneObjects {
		go func(bidder *pbs.PBSBidder, reqJSON bytes.Buffer, mediaType pbs.MediaType) {
			result, err := a.callOne(ctx, reqJSON)
			result.Error = err
			if result.Bid != nil {
				result.Bid.BidderCode = bidder.BidderCode
				result.Bid.BidID = bidder.LookupBidID(result.Bid.AdUnitCode)
				if result.Bid.BidID == "" {
					result.Error = &errortypes.BadServerResponse{
						Message: fmt.Sprintf("Unknown ad unit code '%s'", result.Bid.AdUnitCode),
					}
					result.Bid = nil
				} else {
					// no need to check whether mediaTypes is nil or length of zero, pbs.ParsePBSRequest will cover
					// these cases.
					// for media types other than banner and video, pbs.ParseMediaType will throw error.
					// we may want to create a map/switch cases to support more media types in the future.
					if mediaType == pbs.MEDIA_TYPE_VIDEO {
						result.Bid.CreativeMediaType = string(openrtb_ext.BidTypeVideo)
					} else {
						result.Bid.CreativeMediaType = string(openrtb_ext.BidTypeBanner)
					}
				}
			}
			ch <- result
		}(bidder, obj.requestJson, obj.mediaType)
	}

	bids := make(pbs.PBSBidSlice, 0)
	for i := 0; i < len(callOneObjects); i++ {
		result := <-ch
		if result.Bid != nil && result.Bid.Price != 0 {
			bids = append(bids, result.Bid)
		}
		if req.IsDebug {
			debug := &pbs.BidderDebug{
				RequestURI:   a.URI,
				RequestBody:  callOneObjects[i].requestJson.String(),
				StatusCode:   result.StatusCode,
				ResponseBody: result.ResponseBody,
			}
			bidder.Debug = append(bidder.Debug, debug)
		}
		if result.Error != nil {
			if glog.V(2) {
				glog.Infof("Error from rubicon adapter: %v", result.Error)
			}
			err = result.Error
		}
	}

	if len(bids) == 0 {
		return nil, err
	}
	return bids, nil
}

func appendTrackerToUrl(uri string, tracker string) (res string) {
	// Append integration method. Adapter init happens once
	urlObject, err := url.Parse(uri)
	// No other exception throwing mechanism in this stack, so ignoring parse errors.
	if err == nil {
		values := urlObject.Query()
		values.Add("tk_xint", tracker)
		urlObject.RawQuery = values.Encode()
		res = urlObject.String()
	} else {
		res = uri
	}
	return
}

func NewRubiconAdapter(config *adapters.HTTPAdapterConfig, uri string, xuser string, xpass string, tracker string) *RubiconAdapter {
	return NewRubiconBidder(adapters.NewHTTPAdapter(config).Client, uri, xuser, xpass, tracker)
}

func NewRubiconBidder(client *http.Client, uri string, xuser string, xpass string, tracker string) *RubiconAdapter {
	a := &adapters.HTTPAdapter{Client: client}

	uri = appendTrackerToUrl(uri, tracker)

	return &RubiconAdapter{
		http:         a,
		URI:          uri,
		XAPIUsername: xuser,
		XAPIPassword: xpass,
	}
}

func (a *RubiconAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	numRequests := len(request.Imp)
	errs := make([]error, 0, len(request.Imp))
	var err error

	requestData := make([]*adapters.RequestData, 0, numRequests)
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("User-Agent", "prebid-server/1.0")

	requestImpCopy := request.Imp

	for i := 0; i < numRequests; i++ {
		thisImp := requestImpCopy[i]

		var bidderExt adapters.ExtImpBidder
		if err = json.Unmarshal(thisImp.Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		var rubiconExt openrtb_ext.ExtImpRubicon
		if err = json.Unmarshal(bidderExt.Bidder, &rubiconExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		impExt := rubiconImpExt{
			RP: rubiconImpExtRP{
				ZoneID: rubiconExt.ZoneId,
				Target: rubiconExt.Inventory,
				Track:  rubiconImpExtRPTrack{Mint: "", MintVersion: ""},
			},
		}
		thisImp.Ext, err = json.Marshal(&impExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if request.User != nil {
			userCopy := *request.User
			userExtRP := rubiconUserExt{RP: rubiconUserExtRP{Target: rubiconExt.Visitor}}

			if request.User.Ext != nil {
				var userExt *openrtb_ext.ExtUser
				if err = json.Unmarshal(userCopy.Ext, &userExt); err != nil {
					errs = append(errs, &errortypes.BadInput{
						Message: err.Error(),
					})
					continue
				}
				userExtRP.Consent = userExt.Consent
				if userExt.DigiTrust != nil {
					userExtRP.DigiTrust = userExt.DigiTrust
				}
				userExtRP.Eids = userExt.Eids

				// set user.ext.tpid
				if len(userExt.Eids) > 0 {
					tpIds := make([]rubiconExtUserTpID, 0)
					for _, eid := range userExt.Eids {
						if eid.Source == "adserver.org" {
							uids := eid.Uids
							if len(uids) > 0 {
								uid := uids[0]
								if uid.Ext != nil && uid.Ext.RtiPartner == "TDID" {
									tpIds = append(tpIds, rubiconExtUserTpID{Source: "tdid", UID: uid.ID})
								}
							}
						}
					}
					if len(tpIds) > 0 {
						userExtRP.TpID = tpIds
					}
				}
			}

			userCopy.Ext, err = json.Marshal(&userExtRP)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			request.User = &userCopy
		}

		if request.Device != nil {
			deviceCopy := *request.Device
			deviceExt := rubiconDeviceExt{RP: rubiconDeviceExtRP{PixelRatio: request.Device.PxRatio}}
			deviceCopy.Ext, err = json.Marshal(&deviceExt)
			request.Device = &deviceCopy
		}

		isVideo := isVideo(thisImp)
		if isVideo {
			videoCopy := *thisImp.Video
			videoExt := rubiconVideoExt{Skip: rubiconExt.Video.Skip, SkipDelay: rubiconExt.Video.SkipDelay, RP: rubiconVideoExtRP{SizeID: rubiconExt.Video.VideoSizeID}}
			videoCopy.Ext, err = json.Marshal(&videoExt)
			thisImp.Video = &videoCopy
			thisImp.Banner = nil
		} else {
			primarySizeID, altSizeIDs, err := parseRubiconSizes(thisImp.Banner.Format)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			bannerExt := rubiconBannerExt{RP: rubiconBannerExtRP{SizeID: primarySizeID, AltSizeIDs: altSizeIDs, MIME: "text/html"}}
			bannerCopy := *thisImp.Banner
			bannerCopy.Ext, err = json.Marshal(&bannerExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			thisImp.Banner = &bannerCopy
			thisImp.Video = nil
		}

		siteExt := rubiconSiteExt{RP: rubiconSiteExtRP{SiteID: rubiconExt.SiteId}}
		pubExt := rubiconPubExt{RP: rubiconPubExtRP{AccountID: rubiconExt.AccountId}}

		if request.Site != nil {
			siteCopy := *request.Site
			siteCopy.Ext, err = json.Marshal(&siteExt)
			siteCopy.Publisher = &openrtb.Publisher{}
			siteCopy.Publisher.Ext, err = json.Marshal(&pubExt)
			request.Site = &siteCopy
		}
		if request.App != nil {
			appCopy := *request.App
			appCopy.Ext, err = json.Marshal(&siteExt)
			appCopy.Publisher = &openrtb.Publisher{}
			appCopy.Publisher.Ext, err = json.Marshal(&pubExt)
			request.App = &appCopy
		}

		request.Imp = []openrtb.Imp{thisImp}
		request.Cur = nil
		request.Ext = nil

		reqJSON, err := json.Marshal(request)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		reqData := &adapters.RequestData{
			Method:  "POST",
			Uri:     a.URI,
			Body:    reqJSON,
			Headers: headers,
		}
		reqData.SetBasicAuth(a.XAPIUsername, a.XAPIPassword)
		requestData = append(requestData, reqData)
	}

	return requestData, errs
}

func isVideo(imp openrtb.Imp) bool {
	video := imp.Video
	if video != nil {
		// Do any other media types exist? Or check required video fields.
		return imp.Banner == nil || isFullyPopulatedVideo(video)
	}
	return false
}

func isFullyPopulatedVideo(video *openrtb.Video) bool {
	// These are just recommended video fields for XAPI
	return video.MIMEs != nil && video.Protocols != nil && video.MaxDuration != 0 && video.Linearity != 0 && video.API != nil
}

func (a *RubiconAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}

	var bidReq openrtb.BidRequest
	if err := json.Unmarshal(externalRequest.Body, &bidReq); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	bidType := openrtb_ext.BidTypeBanner

	isVideo := isVideo(bidReq.Imp[0])
	if isVideo {
		bidType = openrtb_ext.BidTypeVideo
	}

	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]
			if bid.Price != 0 {
				// Since Rubicon XAPI returns only one bid per response
				// copy response.bidid to openrtb_response.seatbid.bid.bidid
				if bid.ID == "0" {
					bid.ID = bidResp.BidID
				}
				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     &bid,
					BidType: bidType,
				})
			}
		}
	}

	return bidResponse, nil
}
