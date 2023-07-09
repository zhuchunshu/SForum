<div class="row">
    <div class="col-12 col-lg-4 col-md-6">
        <label class="form-label">是否开启签到奖励</label>
        <div class="divide-y">
            <div>
                <label class="row">
                    <span class="col">签到奖励总开关</span>
                    <span class="col-auto">
                        <label class="form-check form-check-single form-switch">
                            <input class="form-check-input" name="checkin_check" type="checkbox" @if(get_hook_credit_options('checkin_check','true')==="true"){{"checked"}}@endif>
                        </label>
                    </span>
                </label>
            </div>
            <div>
            </div>
        </div>
    </div>

    <div class="col-12 col-lg-4 col-md-6">
        <label class="form-label">是否开启发帖奖励</label>
        <div class="divide-y">
            <div>
                <label class="row">
                    <span class="col">发帖奖励总开关</span>
                    <span class="col-auto">
                        <label class="form-check form-check-single form-switch">
                            <input class="form-check-input" name="create_topic_check" type="checkbox" @if(get_hook_credit_options('create_topic_check','true')==="true"){{"checked"}}@endif>
                        </label>
                    </span>
                </label>
            </div>
            <div>
            </div>
        </div>
    </div>

    <div class="col-12 col-lg-4 col-md-6">
        <label class="form-label">是否开启评论奖励</label>
        <div class="divide-y">
            <div>
                <label class="row">
                    <span class="col">评论奖励总开关</span>
                    <span class="col-auto">
                        <label class="form-check form-check-single form-switch">
                            <input class="form-check-input" name="create_topic_comment_check" type="checkbox" @if(get_hook_credit_options('create_topic_comment_check','true')==="true"){{"checked"}}@endif>
                        </label>
                    </span>
                </label>
            </div>
            <div>
            </div>
        </div>
    </div>

    <div class="col-12 col-lg-4 col-md-6">
        <label class="form-label">是否开启设置头像奖励</label>
        <div class="divide-y">
            <div>
                <label class="row">
                    <span class="col">设置头像奖励总开关</span>
                    <span class="col-auto">
                        <label class="form-check form-check-single form-switch">
                            <input class="form-check-input" name="set_avatar_check" type="checkbox" @if(get_hook_credit_options('set_avatar_check','true')==="true"){{"checked"}}@endif>
                        </label>
                    </span>
                </label>
            </div>
            <div>
            </div>
        </div>
    </div>

    <div class="col-12 col-lg-4 col-md-6">
        <div class="row">
            <div class="col-md-6">
                <label class="form-label">每日发帖奖励数量</label>
                <div class="input-group">
                    <input type="number" name="create_topic_award_number" value="{{get_hook_credit_options('create_topic_award_number',10)}}" class="form-control">
                    <span class="input-group-text">帖</span>
                </div>
            </div>
            <div class="col-md-6">
                <label class="form-label">每日评论奖励数量</label>
                <div class="input-group">
                    <input type="number" name="create_topic_comment_award_number" value="{{get_hook_credit_options('create_topic_comment_award_number',20)}}" class="form-control">
                    <span class="input-group-text">评论</span>
                </div>
            </div>
        </div>
    </div>
</div>