<div class="card-body">
    <div class="mb-3">
        <label for="" class="form-label">商户号 <b class="text-red">*</b></label>
        <input type="text" class="form-control" name="wechat_mch_id" value="{{pay()->get_options('wechat_mch_id')}}">
    </div>
    <div class="mb-3">
        <label for="" class="form-label">商户秘钥 <b class="text-red">*</b></label>
        <input type="text" class="form-control" name="wechat_mch_secret_key" value="{{pay()->get_options('wechat_mch_secret_key')}}">
    </div>
    <div class="mb-3">
        <label for="" class="form-label">商户私钥 <b class="text-red">*</b></label>
        <input type="file" class="form-control" name="wechat_mch_secret_cert">
        <small class="text-muted">{{pay()->get_options('wechat_mch_secret_cert')}}</small>
    </div>
    <div class="mb-3">
        <label for="" class="form-label">商户公钥证书 <b class="text-red">*</b></label>
        <input type="file" class="form-control" name="wechat_mch_public_cert_path">
        <small class="text-muted">{{pay()->get_options('wechat_mch_public_cert_path')}}</small>
    </div>
    <div class="mb-3">
        <label for="" class="form-label">回调地址 <b class="text-red">*</b></label>
        <input type="text" class="form-control" name="wechat_notify_url" value="{{pay()->get_options('wechat_notify_url',url('/api/pay/wechat/notify'))}}">
    </div>
    <div class="mb-3">
        <label for="" class="form-label">公众号/小程序 appid <b class="text-red">*</b></label>
        <input type="text" class="form-control" name="wechat_mp_app_id" value="{{pay()->get_options('wechat_mp_app_id')}}">
    </div>
</div>