@extends('app')
@section('title','网站备份')
@section('content')

    <div class="row row-cards">
        <div class="col-12">
            <div class="card">
                <div class="card-header">
                    <h3 class="card-title">网站备份</h3>
                    <div class="card-actions">
                        <a href="/admin/server/backup/create">创建新的备份</a>
                    </div>
                </div>
                <div class="card-body border-bottom py-3">
                    <div class="table-responsive">
                        <table
                                class="table table-vcenter">
                            <thead>
                            <tr>
                                <th>#</th>
                                <th>文件大小</th>
                                <th>创建时间</th>
                                <th class="w-2"></th>
                                <th class="w-2"></th>
                            </tr>
                            </thead>
                            <tbody>
                            @if(!$page->count())
                                <tr>
                                    <td >无结果</td>
                                    <td class="text-muted" >
                                        无结果
                                    </td>
                                    <td class="text-muted" ><a href="#" class="text-reset">无结果</a></td>
                                    <td>
                                        <a href="#">无结果</a>
                                    </td>
                                    <td>
                                        <a href="#">无结果</a>
                                    </td>
                                </tr>
                            </tbody>
                            @else
                                @foreach($page as $data)
                                    <tr>
                                        <td class="text-muted" >
                                            {{$data['filename']}}
                                        </td>
                                        <td class="text-muted" ><a href="#" class="text-reset">{{round($data['size']/1024/1024,2)}}MB</a></td>
                                        <td class="text-muted">
                                            {{$data['date']}}
                                        </td>
                                        <td class="w-4">
                                            <a href="/admin/server/backup/download?path={{$data['path']}}">下载</a>
                                        </td>
                                        <td class="w-4">
                                            <a href="/admin/server/backup/delete?filename={{$data['filename']}}">删除</a>
                                        </td>
                                    </tr>
                                    @endforeach
                                    @endif
                                    </tbody>
                        </table>
                    </div>
                    <div class="mt-2" style="margin-bottom: -20px">
                        {!! make_page($page) !!}
                    </div>
                </div>
                <div class="card-footer">
                    <span class="text-muted">
                    数据存放位置:{{BASE_PATH."/runtime/backup"}}
                </span>
                </div>
            </div>
        </div>

    </div>

@endsection