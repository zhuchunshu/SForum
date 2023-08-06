<div class="hr-text hr-text-left">其他</div>
<div class="row">


    <div class="col-6 col-lg-3">
        <label for="" class="form-label">qq</label>
        <input class="form-control" type="text" name="options[qq]" value="@if($user->options->qq){{$user->options->qq}}@endif">
    </div>
    <div class="col-6 col-lg-3">
        <label for="" class="form-label">微信</label>
        <input class="form-control" type="text" name="options[wx]" value="@if($user->options->wx){{$user->options->wx}}@endif">
    </div>


    <div class="col-6 col-lg-3">
        <label for="" class="form-label">网站</label>
        <input class="form-control" type="text" name="options[website]" value="@if($user->options->website){{$user->options->website}} @endif">
    </div>

    <div class="col-6 col-lg-3">
        <label for="" class="form-label">对外邮箱</label>
        <input class="form-control" type="text" name="options[email]" value="@if($user->options->email){{$user->options->email}}@endif">
    </div>

    <div class="col-6 col-lg-3">
        <label for="" class="form-label">{{get_options('wealth_credits_name', '积分')}}</label>
        <input class="form-control" type="number" name="options[credits]" value="{{$user->options->credits?:0}}">
    </div>

    <div class="col-6 col-lg-3">
        <label for="" class="form-label">{{get_options('wealth_golds_name', '金币')}}</label>
        <input class="form-control" type="number" min="1" max="999999999" name="options[golds]" value="{{$user->options->golds?:0}}">
    </div>

    <div class="col-6 col-lg-3">
        <label for="" class="form-label">{{get_options('wealth_exp_name', '经验')}}</label>
        <input class="form-control" type="number" name="options[exp]" value="{{$user->options->exp?:0}}">
    </div>

    <div class="col-6 col-lg-3">
        <label for="" class="form-label">{{get_options('wealth_money_name', '余额')}}</label>
        <input class="form-control" min="0.01" step="0.01" type="number" name="options[money]" value="{{$user->options->money?:0}}">
    </div>

    <div class="col-6 col-lg-3">
        <label for="" class="form-label">签名</label>
        <textarea rows="4" name="options[qianming]" class="form-control">{{$user->options->qianming}}</textarea>
    </div>
</div>