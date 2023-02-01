<div class="mb-3">
    <label for="" class="form-label">选择存储服务接口</label>
    <select name="service" class="form-select">
        @foreach((new \App\Plugins\Core\src\Service\FileStoreService())->get_services() as $key=>$data)
            <option value="{{$key}}" @if(get_options('mail_service','b53a68eae9ecac0d86eb8d1125524b13')===$key){{"selected"}} @endif>{{$data['name']}}</option>
        @endforeach
    </select>
</div>