<div id="carousel-captions-{{$id}}" class="carousel slide" data-bs-ride="carousel">
    <div class="carousel-inner {{$class}}" style="{{$style}}">

        @foreach($images as $key => $item)

            @if($loop->first)
                @if(@$item->url)
                    <a href="{{$item->url}}"> @endif
                        <div class="carousel-item active">
                            <img class="d-block w-100" alt="" src="{{$item->image}}"/>
                            @if(@$item->title || @$item->description)
                                <div class="carousel-caption-background d-none d-md-block"></div>
                                <div class="carousel-caption d-none d-md-block">
                                    @if(@$item->title)
                                        <h3>{{$item->title}}</h3>
                                    @endif
                                    @if(@$item->description)
                                        <p>{{$item->description}}</p>
                                    @endif
                                </div>
                            @endif
                        </div>
                        @if(@$item->url) </a>
                @endif

            @else
                @if(@$item->url)
                    <a href="{{$item->url}}"> @endif
                        <div class="carousel-item">
                            <img class="d-block w-100" alt="" src="{{$item->image}}"/>
                            @if(@$item->title || @$item->description)
                                <div class="carousel-caption-background d-none d-md-block"></div>
                                <div class="carousel-caption d-none d-md-block">
                                    @if(@$item->title)
                                        <h3>{{$item->title}}</h3>
                                    @endif
                                    @if(@$item->description)
                                        <p>{{$item->description}}</p>
                                    @endif
                                </div>
                            @endif
                        </div>
                        @if(@$item->url) </a>
                @endif

            @endif

        @endforeach
    </div>
    <a class="carousel-control-prev" data-bs-target="#carousel-captions-{{$id}}" role="button" data-bs-slide="prev">
        <span class="carousel-control-prev-icon" aria-hidden="true"></span>
        <span class="visually-hidden">Previous</span>
    </a>
    <a class="carousel-control-next" data-bs-target="#carousel-captions-{{$id}}" role="button" data-bs-slide="next">
        <span class="carousel-control-next-icon" aria-hidden="true"></span>
        <span class="visually-hidden">Next</span>
    </a>
</div>
