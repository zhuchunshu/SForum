<div class="card-body">
    <div class="mb-3">
        <label for="" class="form-label">支付宝分配的app_id <b class="text-red">*</b></label>
        <input type="text" class="form-control" name="alipay_app_id" value="{{pay()->get_options('alipay_app_id')}}" required>
    </div>
    <div class="mb-3">
        <label for="" class="form-label">应用私钥 <b class="text-red">*</b></label>
        <textarea class="form-control" name="alipay_app_secret_cert" id="" rows="10" required>{{pay()->get_options('alipay_app_secret_cert')}}</textarea>
    </div>
    <div class="mb-3">
        <label for="" class="form-label">应用公钥证书 <b class="text-red">*</b></label>
        <input type="file" class="form-control" name="alipay_app_public_cert_path" value="{{pay()->get_options('alipay_app_public_cert_path')}}" @if(!pay()->get_options('alipay_app_public_cert_path')){{'required'}}@endif>
        <small class="text-muted">{{pay()->get_options('alipay_app_public_cert_path')}}</small>
    </div>
    <div class="mb-3">
        <label for="" class="form-label">支付宝公钥证书 <b class="text-red">*</b></label>
        <input type="file" class="form-control" name="alipay_public_cert_path" value="{{pay()->get_options('alipay_public_cert_path')}}" @if(!pay()->get_options('alipay_public_cert_path')){{'required'}}@endif>
        <small class="text-muted">{{pay()->get_options('alipay_public_cert_path')}}</small>
    </div>
    <div class="mb-3">
        <label for="" class="form-label">支付宝根证书 <b class="text-red">*</b></label>
        <input type="file" class="form-control" name="alipay_root_cert_path" @if(!pay()->get_options('alipay_root_cert_path')){{'required'}}@endif>
        <small class="text-muted">{{pay()->get_options('alipay_root_cert_path')}}</small>
    </div>
    <div class="mb-3">
        <label for="" class="form-label">回调地址 <b class="text-red">*</b></label>
        <input type="text" class="form-control" name="alipay_notify_url" value="{{pay()->get_options('alipay_notify_url',url('/api/pay/alipay/notify'))}}" required>
    </div>
    <div class="mb-3">
        <label for="" class="form-label">支付返回地址 <b class="text-red">*</b></label>
        <input type="text" class="form-control" name="alipay_return_url" value="{{pay()->get_options('alipay_return_url',url('/pay/alipay/return'))}}" required>
    </div>
    <div class="mb-3">
        <label for="" class="form-label">支付方式</label>
        <select name="alipay_pay_mode" class="form-select">
            <option @if (pay()->get_options('alipay_pay_mode','MIX')==='WEB') selected="selected" @endif value="WEB">WEB(电脑端)</option>
            <option @if (pay()->get_options('alipay_pay_mode','MIX')==='WAP') selected="selected" @endif value="WAP">WAP(移动端)</option>
            <option @if (pay()->get_options('alipay_pay_mode','MIX')==='SCAN') selected="selected" @endif value="SCAN">扫码付(当面付)</option>
            <option @if (pay()->get_options('alipay_pay_mode','MIX')==='MIX') selected="selected" @endif value="MIX">混合(WEB+WAP)</option>
        </select>
    </div>
</div>