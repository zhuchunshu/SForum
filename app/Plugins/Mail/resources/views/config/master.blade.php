<div class="mb-3">
    <label for="" class="form-label">选择邮箱发信服务接口</label>
    <select name="service" class="form-select">
        @foreach((new \App\Plugins\Mail\src\Service\SendService())->get_services() as $key=>$data)
            <option value="{{$key}}" @if(get_options('mail_service','ca583971fcbccdf5d7ed77cf4c471ac5')===$key){{"selected"}} @endif>{{$data['name']}}</option>
        @endforeach
    </select>
</div>