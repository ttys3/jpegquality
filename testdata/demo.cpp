//this code from http://hx173149.github.io/2016/04/19/jepge_quality/
//build: g++ demo.cpp -ljpeg
//run: ./a.out
//resut: quality: 75

/*

❯ identify -verbose ./testdata/Landscape_3.jpg | grep -i Quality
  Quality: 73

❯ jhead testdata/Landscape_3.jpg
File name    : testdata/Landscape_3.jpg
File size    : 348796 bytes
File date    : 2019:09:04 17:18:55
Resolution   : 1800 x 1200
Orientation  : rotate 180
JPEG Quality : 73
*/

#include <stdio.h>
#include <math.h>

#include "jpeglib.h"
#include <setjmp.h>

#ifdef _WIN
    #include <stdint.h>
    #define u_int8_t uint8_t
#endif

//标准明度量化表
static const unsigned int std_luminance_quant_tbl[DCTSIZE2] = {
16,  11,  10,  16,  24,  40,  51,  61,
12,  12,  14,  19,  26,  58,  60,  55,
14,  13,  16,  24,  40,  57,  69,  56,
14,  17,  22,  29,  51,  87,  80,  62,
18,  22,  37,  56,  68, 109, 103,  77,
24,  35,  55,  64,  81, 104, 113,  92,
49,  64,  78,  87, 103, 121, 120, 101,
72,  92,  95,  98, 112, 100, 103,  99
};
//读取JPG文件的质量参数
int ReadJpegQuality(const char *filename)
{
  FILE * infile = fopen(filename, "rb");
  fseek(infile,0,SEEK_END);
  size_t sz = ftell(infile);
  fseek(infile,0,SEEK_SET);
  unsigned char* buffer = new unsigned char[sz];
  fread(buffer,1,sz,infile);
  fclose(infile);
  //如果不是JPG格式的文件返回-1
  if(buffer==NULL || sz <= 2 || 
  0xFF != (u_int8_t)buffer[0] || 
  0xD8 != (u_int8_t)buffer[1])
  {
    return -1;
  }
  struct jpeg_decompress_struct cinfo;
  struct jpeg_error_mgr jerr;
  cinfo.err = jpeg_std_error(&jerr);
  jpeg_create_decompress(&cinfo);
  jpeg_mem_src(&cinfo,(unsigned char*)buffer,sz);
  jpeg_read_header(&cinfo, TRUE);
  int tmp_quality = 0;
  int linear_quality = 0;
  const int aver_times = 3;
  int times = 0;
  int aver_quality = 0;
  //量化表反推3次，取平均值
  for(int i=0;i<DCTSIZE2;i++)
  {
    long temp = cinfo.quant_tbl_ptrs[0]->quantval[i];
    if(temp<32767L&&temp>0)
    {
      linear_quality = ceil((float)(temp*100L - 50L)/std_luminance_quant_tbl[i]);
      if(linear_quality==1) tmp_quality = 1;
      else if(linear_quality==100) tmp_quality = 100;
      else if(linear_quality>100)
      {
        tmp_quality = ceil((float)5000/linear_quality);
      }
    else
    {
      tmp_quality = 100 - ceil((float)linear_quality/2);
    }
    aver_quality += tmp_quality;
    if(aver_times==++times)
    {
      aver_quality /= aver_times;
      break;
    } 
    }
  }
  jpeg_destroy_decompress(&cinfo);
  return aver_quality;
}

int main(int argc,char** argv)
{
  printf("quality: %d\n",ReadJpegQuality("./Landscape_3.jpg"));
  return 0; 
}