@extends('App::app')
@section('title','修改文档【 '.$data->name.'】')
@section('content')

    <div class="row row-cards justify-content-center">
        <div class="col-md-12">
            <div class="border-0 card card-body">
                <h3 class="card-title">修改文档 【{{$data->name}}】</h3>
                <form action="/docs/editClass?Redirect=/docs/editClass/{{$data->id}}" method="post" enctype="multipart/form-data">
                    <x-csrf/>
                    <input type="hidden" name="class_id" value="{{$data->id}}">
                    <div class="mb-3">
                        <label class="form-label">文档名</label>
                        <input type="text" value="{{$data->name}}" class="form-control" name="name" required>
                    </div>
                    <div class="mb-3">
                        <label class="form-label">哪些用户组可看?</label>
                        <select name="userClass[]" id="" class="form-select" multiple>
                           @if($data->quanxian!=="null")
                                @php $quanxian = json_decode($data->quanxian) @endphp
                                @foreach($userClass as $value)
                                    @if(in_array($value->id,$quanxian))
                                        <option value="{{$value->id}}" selected>{{$value->name}} 「权限值:{{$value['permission-value']}}」</option>
                                    @else
                                        <option value="{{$value->id}}">{{$value->name}} 「权限值:{{$value['permission-value']}}」</option>
                                    @endif
                                @endforeach
                            @else
                                @foreach($userClass as $data)
                                    <option value="{{$data->id}}">{{$data->name}} 「权限值:{{$data['permission-value']}}」
                                    </option>
                                @endforeach
                            @endif
                        </select>
                    </div>
                    <div class="mb-3">
                        <label class="form-check form-switch">
                            @if((bool)$data->public===true)
                                <input name="public" class="form-check-input"
                                       type="checkbox" checked>
                            @else
                                <input name="public" class="form-check-input"
                                       type="checkbox">
                            @endif
                            <span
                                    class="form-check-label">允许所有人可看</span>
                        </label>
                    </div>
                    <div class="mb-3">
                        <button class="btn btn-primary" type="submit">提交</button>
                    </div>
                </form>
            </div>
        </div>
    </div>



@endsection

@section('header')
    <div class="page-wrapper">
        <div class="container-xl">
            <!-- Page title -->
            <div class="page-header d-print-none">
                <div class="row align-items-center">
                    <div class="col">
                        <!-- Page pre-title -->
                        <div class="page-pretitle">
                            Overview
                        </div>
                        <h2 class="page-title">
                            修改文档 【{{$data->name}}】
                        </h2>
                    </div>


                </div>
            </div>
        </div>
    </div>
@endsection