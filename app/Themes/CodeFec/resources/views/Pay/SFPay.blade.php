<div class="card-body">
    <div class="mb-3">
        <label for="" class="form-label">回调地址 <b class="text-red">*</b></label>
        <input type="text" class="form-control" name="sfpay_notify_url" value="{{pay()->get_options('sfpay_notify_url',url('/api/pay/SFPay/notify'))}}" required>
    </div>
</div>