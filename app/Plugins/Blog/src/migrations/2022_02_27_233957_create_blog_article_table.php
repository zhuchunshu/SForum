<?php

use Hyperf\Database\Schema\Schema;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Migrations\Migration;

class CreateBlogArticleTable extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::create('blog_article', function (Blueprint $table) {
            $table->bigIncrements('id');
			$table->string('blog_id');
			$table->text('title');
			$table->string('class_id');
			$table->longText('content');
			$table->longText('markdown');
            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('blog_article');
    }
}
