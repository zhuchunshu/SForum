<div class="card card-body">

    <div class="mb-3">
        <div class="row">
            <div class="col-3">
                <label class="form-label">强制短信验证</label>
                <select class="form-select" v-model="data.core_user_sms">
                    <option value="开启">开启</option>
                    <option value="关闭">关闭</option>
                </select>
                <small>默认:关闭</small>
            </div>
            <div class="col-3">
                <label class="form-label">SMS服务商</label>
                <select class="form-select" v-model="data.core_user_sms_service">
                    @foreach(Itf()->get('SMS') as $k=>$v)
                        <option value="{{$v['name']}}">{{$v['name']}}</option>
                    @endforeach
                </select>
                <small>当前:{{get_options('core_user_sms_service','Qcloud')}}</small>
            </div>

            <div class="col-3">
                <label class="form-label">发信限制</label>
                <input type="number" min="1" class="form-control" v-model="data.core_user_sms_limit">
                <small>当前每位用户每日只能发 {{get_options('core_user_sms_limit',1)}} 条，超出收费</small>
            </div>

            <div class="col-3">
                <label class="form-label">短信单价 /分</label>
                <input type="number" min="1" class="form-control" v-model="data.core_user_sms_price">
                <small>当前: {{get_options('core_user_sms_price',1)}}</small>
            </div>
        </div>
    </div>



    @foreach(Itf()->get('SMS') as $k=>$v)
        <div class="mb-3">
            @include($v['view'])
        </div>
    @endforeach


</div>