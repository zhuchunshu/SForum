<div class="mb-3">
    <label class="form-label">设置启用接口</label>
    <div class="form-selectgroup form-selectgroup-boxes d-flex flex-column">
        @foreach((new \App\Plugins\User\src\Service\Oauth2())->get_all() as $data)
        <label class="form-selectgroup-item flex-fill">
                <input type="checkbox" name="enable[]" value="{{$data['mark']}}" class="form-selectgroup-input" @if(\App\Plugins\User\src\Service\oauth2()->check_enable($data['mark'])) checked @endif >
                <div class="form-selectgroup-label d-flex align-items-center p-3">
                    <div class="me-3">
                        <span class="form-selectgroup-check"></span>
                    </div>
                    <div class="form-selectgroup-label-content d-flex align-items-center">
                        <span class="avatar me-3">{!! $data['icon'] !!}</span>
                        <div>
                            <div class="font-weight-medium">{{$data['mark']}}</div>
                            <div class="text-muted">{{$data['name']}}</div>
                        </div>
                    </div>
                </div>
            </label>
        @endforeach
    </div>
</div>