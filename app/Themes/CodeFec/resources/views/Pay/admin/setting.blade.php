@extends("app")

@section('title',"支付设置")

@section('content')
    <form action="/admin/Pay/setting" method="post">
        <x-csrf/>
        <div class="row row-cards">
            <div class="col-md-12">
                <div class="card">
                    <div class="card-body">
                        <div class="mb-3">
                            <label class="form-label">启用支付插件</label>
                            <div class="form-selectgroup form-selectgroup-boxes d-flex flex-column">
                                @foreach(pay()->getInterfaces() as $pay)
                                    <label class="form-selectgroup-item flex-fill">
                                        <input type="checkbox" name="pay[]" value="{{$pay['ename']}}"
                                               class="form-selectgroup-input" @if(in_array($pay['ename'],pay()->get_enabled())) checked="checked" @endif>
                                        <div class="form-selectgroup-label d-flex align-items-center p-3">
                                            <div class="me-3">
                                                <span class="form-selectgroup-check"></span>
                                            </div>
                                            <div class="form-selectgroup-label-content d-flex align-items-center">

                                                <img style="max-height:35px;width:75px;" src="{{$pay['icon']}}" alt="{{$pay['name']}}">
                                                <div>
                                                    <div class="font-weight-medium">{{$pay['name']}}</div>
                                                    <div class="text-muted">{{$pay['description']}}</div>
                                                </div>
                                            </div>
                                        </div>
                                    </label>
                                @endforeach
                            </div>
                        </div>
                    </div>
                </div>
            </div>
            <div class="col-md-12">
                <button @@click="submit" class="btn btn-light">提交</button>
            </div>
        </div>
    </form>
@endsection

@section('scripts')
    <script src="{{ mix("js/admin/pay.js") }}"></script>
@endsection