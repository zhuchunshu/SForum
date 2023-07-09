<div class="row">
{{--    积分--}}
    <div class="col-12 col-md-6">
        <label for="" class="form-label">
            设置头像奖励{{get_options('wealth_credit_name','积分')}}
        </label>
        <input type="number" class="form-control" name="credit_set_avatar"
               value="{{get_hook_credit_options('credit_set_avatar',100)}}">
    </div>

    <div class="col-12 col-md-6">
        <label class="form-label">是否开启设置头像{{get_options('wealth_credit_name','积分'),}}奖励</label>
        <div class="divide-y">
            <div>
                <label class="row">
                    <span class="col">开启设置头像{{get_options('wealth_credit_name','积分'),}}奖励</span>
                    <span class="col-auto">
                        <label class="form-check form-check-single form-switch">
                            <input class="form-check-input" name="credit_set_avatar_checkbox" type="checkbox" @if(get_hook_credit_options('credit_set_avatar_checkbox','true')==="true"){{"checked"}}@endif>
                        </label>
                    </span>
                </label>
            </div>
            <div>
            </div>
        </div>

    </div>
{{--金币--}}
    <div class="col-12 col-md-6">
        <label for="" class="form-label">
            设置头像奖励{{get_options('wealth_golds_name','金币')}}
        </label>
        <input type="number" class="form-control" name="golds_set_avatar"
               value="{{get_hook_credit_options('golds_set_avatar',100)}}">
    </div>

    <div class="col-12 col-md-6">
        <label class="form-label">是否开启设置头像{{get_options('wealth_golds_name','金币'),}}奖励</label>
        <div class="divide-y">
            <div>
                <label class="row">
                    <span class="col">开启设置头像{{get_options('wealth_golds_name','金币'),}}奖励</span>
                    <span class="col-auto">
                        <label class="form-check form-check-single form-switch">
                            <input class="form-check-input" name="golds_set_avatar_checkbox" type="checkbox" @if(get_hook_credit_options('golds_set_avatar_checkbox')==="true"){{"checked"}}@endif>
                        </label>
                    </span>
                </label>
            </div>
            <div>
            </div>
        </div>

    </div>

</div>