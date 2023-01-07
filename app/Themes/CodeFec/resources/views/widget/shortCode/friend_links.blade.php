@php($results = \App\Plugins\Core\src\Models\FriendLink::query()->orderByDesc('to_sort')->get())
<div class="card">
    <div class="card-table table-responsive">
        <table class="table table-vcenter">
            <thead>
            <tr>
                <th>图标</th>
                <th>名称</th>
                <th>网址</th>
                <th>描述</th>
            </tr>
            </thead>
            @if(!\App\Plugins\Core\src\Models\FriendLink::query()->count())
                <tr>
                    <td>
                        暂无结果
                        <a href="#" class="ms-1" aria-label="Open website">
                            <!-- Download SVG icon from http://tabler-icons.io/i/link -->
                            <svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24" viewBox="0 0 24 24"
                                 stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round"
                                 stroke-linejoin="round">
                                <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                                <path d="M10 14a3.5 3.5 0 0 0 5 0l4 -4a3.5 3.5 0 0 0 -5 -5l-.5 .5"/>
                                <path d="M14 10a3.5 3.5 0 0 0 -5 0l-4 4a3.5 3.5 0 0 0 5 5l.5 -.5"/>
                            </svg>
                        </a>
                    </td>
                    <td class="text-muted">暂无结果</td>
                    <td class="text-muted">暂无结果</td>
                    <td class="text-muted">暂无结果</td>
                </tr>
            @else
                @foreach($results as $data)
                    <tr>
                        <td>
                            @if($data->icon)
                                <span class="avatar avatar-sm" style="background-image: url({{$data->icon}})"></span>
                            @else
                                暂无
                            @endif
                        </td>
                        <td>
                            {{$data->name}}
                        </td>
                        <td>
                            <a href="{{$data->link}}" target="_blank" class="ms-1" aria-label="Open website">
                                <!-- Download SVG icon from http://tabler-icons.io/i/link -->
                                {{$data->link}}
                                <svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24" viewBox="0 0 24 24"
                                                                                 stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round"
                                                                                 stroke-linejoin="round">
                                    <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                                    <path d="M10 14a3.5 3.5 0 0 0 5 0l4 -4a3.5 3.5 0 0 0 -5 -5l-.5 .5"/>
                                    <path d="M14 10a3.5 3.5 0 0 0 -5 0l-4 4a3.5 3.5 0 0 0 5 5l.5 -.5"/>
                                </svg>
                            </a>
                        </td>
                        <td class="text-muted">
                            @if($data->description)
                                {{$data->description}}
                            @else
                                暂无结果
                            @endif
                        </td>
                    </tr>
                @endforeach
            @endif

        </table>
    </div>
</div>
