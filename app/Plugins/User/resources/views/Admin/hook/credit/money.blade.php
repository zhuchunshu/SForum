<div class="row">
    {{--    <div class="col-6 col-md-4 col-lg-3">--}}
    {{--    签到随机奖励--}}
    <div class="col-12 col-md-6">
        <label for="" class="form-label">
            签到奖励随机{{get_options('wealth_money_name','余额')}}
        </label>
        <div class="row">
            <div class="col">
                <input type="number" class="form-control" name="money_checkin_min"
                       value="{{get_hook_credit_options('money_checkin_min',10)}}">
                <small>最少</small>
            </div>
            <div class="col">
                <input type="number" class="form-control" name="money_checkin_max"
                       value="{{get_hook_credit_options('money_checkin_max',100)}}">
                <small>最多</small>
            </div>
        </div>
    </div>

    <div class="col-12 col-md-6">
        <label class="form-label">是否开启签到随机{{get_options('wealth_money_name','余额'),}}奖励</label>
        <div class="divide-y">
            <div>
                <label class="row">
                    <span class="col">开启签到随机{{get_options('wealth_money_name','余额'),}}奖励</span>
                    <span class="col-auto">
                        <label class="form-check form-check-single form-switch">
                            <input class="form-check-input" name="money_checkin_checkbox" type="checkbox" @if(get_hook_credit_options('money_checkin_checkbox','true')==="true"){{"checked"}}@endif>
                        </label>
                    </span>
                </label>
            </div>
            <div>
            </div>
        </div>

    </div>

    {{--    签到固定奖励--}}
    <div class="col-12 col-md-6">
        <label for="" class="form-label">
            签到奖励固定{{get_options('wealth_money_name','余额')}}
        </label>
        <div class="row">
            <input type="number" class="form-control" name="money_checkin_fix"
                   value="{{get_hook_credit_options('money_checkin_fix',10)}}">
        </div>
    </div>

    <div class="col-12 col-md-6">
        <label class="form-label">是否开启签到固定{{get_options('wealth_money_name','余额'),}}奖励</label>
        <div class="divide-y">
            <div>
                <label class="row">
                    <span class="col">开启签到固定{{get_options('wealth_money_name','余额'),}}奖励  </span>
                    <span class="col-auto">
                        <label class="form-check form-check-single form-switch">
                            <input class="form-check-input" name="money_checkin_fix_checkbox" type="checkbox" @if(get_hook_credit_options('money_checkin_fix_checkbox','false')==="true"){{"checked"}}@endif>
                        </label>
                    </span>
                </label>
            </div>
            <div>
            </div>
        </div>

    </div>

    {{--    发帖随机奖励--}}
    <div class="col-12 col-md-6">
        <label for="" class="form-label">
            发帖奖励随机{{get_options('wealth_money_name','余额')}}
        </label>
        <div class="row">
            <div class="col">
                <input type="number" class="form-control" name="money_create_topic_min"
                       value="{{get_hook_credit_options('money_create_topic_min',10)}}">
                <small>最少</small>
            </div>
            <div class="col">
                <input type="number" class="form-control" name="money_create_topic_max"
                       value="{{get_hook_credit_options('money_create_topic_max',100)}}">
                <small>最多</small>
            </div>
        </div>
    </div>

    <div class="col-12 col-md-6">
        <label class="form-label">是否开启发帖随机{{get_options('wealth_money_name','余额'),}}奖励</label>
        <div class="divide-y">
            <div>
                <label class="row">
                    <span class="col">开启发帖随机{{get_options('wealth_money_name','余额'),}}奖励  </span>
                    <span class="col-auto">
                        <label class="form-check form-check-single form-switch">
                            <input class="form-check-input" name="money_create_topic_checkbox" type="checkbox" @if(get_hook_credit_options('money_create_topic_checkbox','false')==="true"){{"checked"}}@endif>
                        </label>
                    </span>
                </label>
            </div>
            <div>
            </div>
        </div>

    </div>

    {{--    发帖固定奖励--}}
    <div class="col-12 col-md-6">
        <label for="" class="form-label">
            发帖奖励固定{{get_options('wealth_money_name','余额')}}
        </label>
        <div class="row">
            <input type="number" class="form-control" name="money_create_topic_min"
                   value="{{get_hook_credit_options('money_create_topic_fix',10)}}">
        </div>
    </div>

    <div class="col-12 col-md-6">
        <label class="form-label">是否开启发帖固定{{get_options('wealth_money_name','余额'),}}奖励</label>
        <div class="divide-y">
            <div>
                <label class="row">
                    <span class="col">开启发帖固定{{get_options('wealth_money_name','余额'),}}奖励  </span>
                    <span class="col-auto">
                        <label class="form-check form-check-single form-switch">
                            <input class="form-check-input" name="money_create_topic_fix_checkbox" type="checkbox" @if(get_hook_credit_options('money_create_topic_fix_checkbox','false')==="true"){{"checked"}}@endif>
                        </label>
                    </span>
                </label>
            </div>
            <div>
            </div>
        </div>

    </div>
    {{--    评论随机奖励--}}
    <div class="col-12 col-md-6">
        <label for="" class="form-label">
            评论奖励随机{{get_options('wealth_money_name','余额')}}
        </label>
        <div class="row">
            <div class="col">
                <input type="number" class="form-control" name="money_create_topic_min"
                       value="{{get_hook_credit_options('money_create_topic_comment_min',10)}}">
                <small>最少</small>
            </div>
            <div class="col">
                <input type="number" class="form-control" name="money_create_topic_max"
                       value="{{get_hook_credit_options('money_create_topic_comment_max',100)}}">
                <small>最多</small>
            </div>
        </div>
    </div>

    <div class="col-12 col-md-6">
        <label class="form-label">是否开启评论随机{{get_options('wealth_money_name','余额'),}}奖励</label>
        <div class="divide-y">
            <div>
                <label class="row">
                    <span class="col">开启评论随机{{get_options('wealth_money_name','余额'),}}奖励  </span>
                    <span class="col-auto">
                        <label class="form-check form-check-single form-switch">
                            <input class="form-check-input" name="money_create_topic_comment_checkbox" type="checkbox" @if(get_hook_credit_options('money_create_topic_checkbox','false')==="true"){{"checked"}}@endif>
                        </label>
                    </span>
                </label>
            </div>
            <div>
            </div>
        </div>

    </div>

    {{--    发帖固定奖励--}}
    <div class="col-12 col-md-6">
        <label for="" class="form-label">
            发帖奖励固定{{get_options('wealth_money_name','余额')}}
        </label>
        <div class="row">
            <input type="number" class="form-control" name="money_create_topic_min"
                   value="{{get_hook_credit_options('money_create_topic_comment_fix',10)}}">
        </div>
    </div>

    <div class="col-12 col-md-6">
        <label class="form-label">是否开启发帖固定{{get_options('wealth_money_name','余额'),}}奖励</label>
        <div class="divide-y">
            <div>
                <label class="row">
                    <span class="col">开启发帖固定{{get_options('wealth_money_name','余额'),}}奖励  </span>
                    <span class="col-auto">
                        <label class="form-check form-check-single form-switch">
                            <input class="form-check-input" name="money_create_topic_comment_fix_checkbox" type="checkbox" @if(get_hook_credit_options('money_create_topic_comment_fix_checkbox','false')==="true"){{"checked"}}@endif>
                        </label>
                    </span>
                </label>
            </div>
            <div>
            </div>
        </div>

    </div>
</div>